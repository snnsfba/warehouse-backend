package cache

import (
	"context"
	"data-service/internal/models"
	"data-service/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedProductRepository struct {
	realRepo repository.ProductRepository
	redis    *redis.Client
	ttl      time.Duration
}

func NewCachedProductRepository(realRepo repository.ProductRepository, redis *redis.Client) *CachedProductRepository {
	return &CachedProductRepository{
		realRepo: realRepo,
		redis:    redis,
		ttl:      5 * time.Minute,
	}
}

func (c *CachedProductRepository) GetByID(ctx context.Context, id int) (*models.Product, error) {
	key := fmt.Sprintf("product:%d", id)

	data, err := c.redis.Get(ctx, key).Bytes()

	switch {
	case err == nil:
		if string(data) == "notfound" {
			return nil, repository.ErrNotFound
		}

		var product models.Product
		if err := json.Unmarshal(data, &product); err != nil {
			log.Printf("Failed to unmarshal cached product (continuing with DB): %v", err)
			break
		}

		return &product, nil

	case errors.Is(err, redis.Nil):

	default:
		log.Printf("Redis error (continuing with DB): %v", err)
	}

	product, err := c.realRepo.GetByID(ctx, id)
	if err != nil {
		if setErr := c.redis.Set(ctx, key, "notfound", 1*time.Minute).Err(); setErr != nil {
			log.Printf("Failed to cache notfound: %v", setErr)
		}
		return nil, err
	}

	jsonData, err := json.Marshal(product)
	if err != nil {
		log.Printf("Failed to marshal product: %v", err)
		return product, nil
	}

	if err := c.redis.Set(ctx, key, jsonData, c.ttl).Err(); err != nil {
		log.Printf("failed to cache product: %v", err)
	}

	return product, nil
}

func (c *CachedProductRepository) invalidateProductCache(ctx context.Context, productID int, category string) {
	productKey := fmt.Sprintf("product:%d", productID)

	if err := c.redis.Del(ctx, productKey).Err(); err != nil {
		log.Printf("Failed to delete product cache %s: %v", productKey, err)
	}

	if err := c.redis.Del(ctx, "products:all").Err(); err != nil {
		log.Printf("Failed to delete products:all cache: %v", err)
	}

	if category != "" {
		categoryKey := fmt.Sprintf("products:category:%s", category)
		if err := c.redis.Del(ctx, categoryKey).Err(); err != nil {
			log.Printf("Failed to delete category cache %s: %v", categoryKey, err)
		}

	}

}

func (c *CachedProductRepository) invalidateCategoryCache(ctx context.Context, category string) {
	if category != "" {
		categoryKey := fmt.Sprintf("products:category:%s", category)
		err := c.redis.Del(ctx, categoryKey).Err()
		if err != nil {
			log.Printf("Failed to delete category cache %s: %v", categoryKey, err)
		}

	}

}

func (c *CachedProductRepository) Update(ctx context.Context, product *models.Product) error {
	oldProduct, err := c.realRepo.GetByID(ctx, product.ProductID)
	if err != nil {
		c.invalidateProductCache(ctx, product.ProductID, "")
		return err
	}
	c.invalidateProductCache(ctx, product.ProductID, oldProduct.Category)

	if oldProduct.Category != product.Category {
		c.invalidateCategoryCache(ctx, product.Category)
	}

	return c.realRepo.Update(ctx, product)
}

func (c *CachedProductRepository) Create(ctx context.Context, product *models.Product) error {
	if err := c.redis.Del(ctx, "products:all").Err(); err != nil {
		log.Printf("Failed to delete product cache: %v", err)
	}

	if product.Category != "" {
		categoryKey := fmt.Sprintf("products:category:%s", product.Category)
		if err := c.redis.Del(ctx, categoryKey).Err(); err != nil {
			log.Printf("failed to delete category cache: %v", err)
		}
	}

	return c.realRepo.Create(ctx, product)
}

func (c *CachedProductRepository) Delete(ctx context.Context, id int) error {
	product, err := c.realRepo.GetByID(ctx, id)
	if err != nil {
		c.invalidateProductCache(ctx, id, "")
		return err
	}

	c.invalidateProductCache(ctx, id, product.Category)

	return c.realRepo.Delete(ctx, id)
}

func (c *CachedProductRepository) GetAll(ctx context.Context) ([]models.Product, error) {
	key := "products:all"

	data, err := c.redis.Get(ctx, key).Bytes()

	if err == nil {
		var products []models.Product
		json.Unmarshal(data, &products)
		return products, nil
	}

	if err != redis.Nil {
		log.Printf("Redis error: %v (continuing with DB)", err)
	}

	products, err := c.realRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(products)
	if err != nil {
		log.Printf("failed to marshal products: %v", err)
	} else {
		c.redis.Set(ctx, key, jsonData, c.ttl)
	}

	return products, nil
}

func (c *CachedProductRepository) GetByCategory(ctx context.Context, category string) ([]models.Product, error) {
	key := fmt.Sprintf("products:category:%s", category)

	data, err := c.redis.Get(ctx, key).Bytes()

	if err == nil {
		var products []models.Product
		json.Unmarshal(data, &products)
		return products, nil
	}

	if err != redis.Nil {
		log.Printf("Redis error: %v (continuing with DB)", err)
	}

	products, err := c.realRepo.GetByCategory(ctx, category)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(products)
	if err != nil {
		log.Printf("failed to marshall products: %v", err)
	} else {
		c.redis.Set(ctx, key, jsonData, c.ttl)
	}

	return products, nil
}

func (c *CachedProductRepository) UpdateQuantity(ctx context.Context, id int, change int) error {
	product, err := c.realRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	product.Quantity += change
	return c.Update(ctx, product)
}
