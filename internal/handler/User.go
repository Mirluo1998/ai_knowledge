package handler

import (
	"encoding/json"
	"io"
	"knowledge/internal/model"
	"knowledge/internal/service"
	"log/slog"
	"net/http"
	"strconv"
)

type UserHandler struct {
	logger *slog.Logger
	svc    *service.UserService
}

func NewUserHandler(logger *slog.Logger, svc *service.UserService) *UserHandler {
	return &UserHandler{logger: logger, svc: svc}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	var query model.UserQuery
	id := r.URL.Query().Get("id")
	if id != "" {
		idInt64, _ := strconv.ParseInt(id, 10, 64)
		query.ID = idInt64
	}
	query.Username = r.URL.Query().Get("username")
	query.Email = r.URL.Query().Get("email")
	resp, err := h.svc.GetUser(r.Context(), query)
	var result Result
	if err != nil {
		result = Fail(err.Error())
	} else {
		result = Success(resp)
	}
	if resultJson, err := json.Marshal(result); err != nil {
		h.logger.Error("marshal result failed", "error", err)
	} else {
		w.Write(resultJson)
	}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var registerUser model.User
	buf, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeResult(w, Fail(err.Error()))
		return
	}
	if err := json.Unmarshal(buf, &registerUser); err != nil {
		writeResult(w, Fail(err.Error()))
		return
	}
	_, err = h.svc.RegisterUser(r.Context(), registerUser)
	if err != nil {
		writeResult(w, Fail(err.Error()))
		return
	}
	writeResult(w, Success("User registered successfully"))
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginUser model.User
	buf := make([]byte, 1024)
	buf, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeResult(w, Fail(err.Error()))
		return
	}
	if err := json.Unmarshal(buf, &loginUser); err != nil {
		writeResult(w, Fail(err.Error()))
		return
	}
	_, err = h.svc.Login(r.Context(), loginUser)
	if err != nil {
		writeResult(w, Fail(err.Error()))
		return
	}
	writeResult(w, Success("Login successful"))
}

func writeResult(w http.ResponseWriter, result Result) {
	marshal, err := json.Marshal(result)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(marshal)
}
