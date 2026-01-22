package handlers

import (
	"data-service/internal/models"
	"data-service/internal/repository"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	repo repository.ProductRepository
}

func NewProductHandler(repo repository.ProductRepository) *ProductHandler {
	return &ProductHandler{repo: repo}
}

type ProductCreateRequest struct {
	Price       float64 `json:"price"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	Category    string  `json:"category"`
}

type ProductUpdateRequest struct {
	Price       float64 `json:"price"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	Category    string  `json:"category"`
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid product id", nil)
		return
	}

	product, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "product not found", nil)
		case errors.Is(err, repository.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error(), nil)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get product", nil)
		}
		return
	}

	writeJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.GetAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get products", nil)
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) GetByCategory(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	if category == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "category is required", nil)
		return
	}

	products, err := h.repo.GetByCategory(r.Context(), category)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error(), nil)
		default:
			writeError(w, http.StatusInternalServerError, "invalid_input", err.Error(), nil)
		}
		return
	}

	writeJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ProductCreateRequest
	if ok := decodeJSON(w, r, &req); !ok {
		return
	}

	p := models.Product{
		Name:        req.Name,
		Price:       req.Price,
		Description: req.Description,
		Quantity:    req.Quantity,
		Category:    req.Category,
	}

	if err := h.repo.Create(r.Context(), &p); err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error(), nil)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create product", nil)
		}
		return
	}

	w.Header().Set("Location", "/products/"+strconv.Itoa(p.ProductID))
	writeJSON(w, http.StatusCreated, p)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid product id", nil)
		return
	}

	var req ProductUpdateRequest
	if ok := decodeJSON(w, r, &req); !ok {
		return
	}

	p := models.Product{
		ProductID:   id,
		Name:        req.Name,
		Price:       req.Price,
		Description: req.Description,
		Quantity:    req.Quantity,
		Category:    req.Category,
	}

	if err := h.repo.Update(r.Context(), &p); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "product not found", nil)
		case errors.Is(err, repository.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error(), nil)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to update product", nil)
		}
		return

	}

	writeJSON(w, http.StatusOK, p)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_id", "invalid product id", nil)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found", "product not found", nil)
		case errors.Is(err, repository.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid_input", err.Error(), nil)
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete product", nil)
		}
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}
