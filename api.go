package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

// WriteJSON writes the given interface as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

// const jwtSecret = "GoBank999"

// func createJWT(account *Account) (string, error) {
// 	claims := &jwt.MapClaims{
// 		"expiresAt":     15000,
// 		"accountNumber": account.Number,
// 	}
// 	secret := os.Getenv("JWT_SECRET")
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString([]byte(secret))
// }

func permissionDenied(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, ApiError{Error: "Pemission Denied"})
}

func (s *APIServer) handleLoan(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method %s not allowed", r.Method)
	}

	var req LoanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			return fmt.Errorf("empty request body")
		}
		if err == io.ErrUnexpectedEOF {
			return fmt.Errorf("unexpected end of JSON input")
		}
		return err
	}

	if req.UserID <= 0 {
		return fmt.Errorf("user id cannot be empty")
	}
	if req.Plafond <= 0 {
		return fmt.Errorf("plafond cannot be empty")
	}
	if req.Duration <= 0 {
		return fmt.Errorf("loan duration cannot be empty")
	}
	if req.Fees <= 0 {
		return fmt.Errorf("fees cannot be empty")
	}
	if req.StartDate == "" {
		return fmt.Errorf("start date cannot be empty")
	}

	loan, err := NewLoan(req.UserID, req.Plafond, req.Duration, req.Fees, req.StartDate)
	if err != nil {
		return err
	}

	if err := s.store.CreateLoan(loan); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, loan)
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("withJWTAuth")
		tokenString := r.Header.Get("x-jwt-token")

		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}

		token, err := validateJWT(tokenString)
		if err != nil {
			permissionDenied(w)
			return
		}

		if !token.Valid {
			permissionDenied(w)
			return
		}

		userID, err := getID(r)
		if err != nil {
			permissionDenied(w)
			return
		}
		account, err := s.GetAccountById(userID)
		if err != nil {
			permissionDenied(w)
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		fmt.Println(claims)
		claimsAccountID, ok := claims["account_id"].(float64)
		if !ok {
			permissionDenied(w)
			return
		}
		accountID := int(claimsAccountID)
		if accountID != account.ID {
			permissionDenied(w)
			return
		}
		handlerFunc(w, r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct {
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

type APIServer struct {
	listenAddr string
	store      Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() error {
	router := mux.NewRouter()

	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	router.HandleFunc("/loan/simulation", makeHTTPHandleFunc(s.handleLoanSimulation))
	router.HandleFunc("/loan", makeHTTPHandleFunc(s.handleLoan))
	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))

	log.Println("JSON API server is running at ", s.listenAddr)
	http.ListenAndServe(s.listenAddr, router)
	return nil
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method %s not allowed", r.Method)
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}
	if req.Email == "" || req.Password == "" {
		return fmt.Errorf("email and password are required")
	}
	account, err := s.store.LoginAccount(req.Email, req.Password)
	if err != nil {
		return err
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["account_id"] = account.ID
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	secret := os.Getenv("JWT_SECRET")
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return err
	}

	resp := map[string]interface{}{
		"account": account,
		"token":   tokenString,
	}

	return WriteJSON(w, http.StatusOK, resp)
}

func (s *APIServer) handleLoanSimulation(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		return fmt.Errorf("method %s not allowed", r.Method)
	}

	var req LoanRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	// Format yang sesuai dengan string tanggal
	layout := "2006-01-02"

	// Memparsing string tanggal ke dalam tipe time.Time
	parsedDate, err := time.Parse(layout, req.StartDate)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return err
	}

	if req.Plafond <= 0 {
		return fmt.Errorf("plafond cannot be empty")
	}
	if req.Duration <= 0 {
		return fmt.Errorf("loan duration cannot be empty")
	}
	if req.Fees <= 0 {
		return fmt.Errorf("fees cannot be empty")
	}

	plafond := req.Plafond
	loanDuration := req.Duration
	fees := req.Fees

	feesPercentage := (fees / 12) / 100
	valA := (plafond * feesPercentage) * (math.Pow((1+feesPercentage), 12) / (math.Pow((1+feesPercentage), 12) - 1))
	valA = math.Round(valA*100) / 100
	loanDetails := make([]map[string]interface{}, int(loanDuration))
	for i := 0; i < int(loanDuration); i++ {

		valB := (feesPercentage / 360) * (loanDuration * 30)
		var vap float64 = 0
		if i > 0 {
			vap = loanDetails[i-1]["Sisa angsuran pokok"].(float64)
		}

		var valSap float64 = 0
		if vap != 0 {
			valB *= math.Round(loanDetails[i-1]["Sisa angsuran pokok"].(float64))
			valSap = loanDetails[i-1]["Sisa angsuran pokok"].(float64) - (valA - valB)
		} else {
			valB *= plafond
			valSap = plafond - (valA - valB*1)
		}
		valB = math.Round(valB)

		loanDetails[i] = map[string]interface{}{
			"id":                  i + 1,
			"tanggal":             parsedDate.AddDate(0, i, 0).Format("2006-01-02"),
			"Angsuran pokok":      math.Round(valA) - valB,
			"Angsuran bunga":      valB,
			"Total angsuran":      valA,
			"Sisa angsuran pokok": valSap,
			"sapV":                math.Round(valSap),
		}
	}

	for i := range loanDetails {
		loanDetails[i]["Sisa angsuran pokok"] = loanDetails[i]["sapV"]
		if loanDetails[i]["Sisa angsuran pokok"].(float64) <= 0 {
			loanDetails[i]["Sisa angsuran pokok"] = 0
		}
		delete(loanDetails[i], "sapV")
	}
	return WriteJSON(w, http.StatusOK, loanDetails)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}
	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}
	return fmt.Errorf("method %s not allowed", r.Method)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		id, err := getID(r)
		if err != nil {
			return err
		}

		account, err := s.store.GetAccountById(id)
		if err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, account)
	}

	if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	req := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	if req.Email == "" || req.Password == "" {
		return fmt.Errorf("email and password are required")
	}
	fmt.Println(req)
	account, err := NewAccount(req.Firstname, req.Lastname, req.Email, req.Password)
	if err != nil {
		return err
	}

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getID(r)
	if err != nil {
		return err
	}
	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	transferReq := new(TransferRequest)

	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}
	defer r.Body.Close()
	return WriteJSON(w, http.StatusOK, transferReq)
}

func getID(r *http.Request) (int, error) {
	fmt.Println(mux.Vars(r)["id"])
	var idStr string
	if mux.Vars(r)["id"] == "" {
		fmt.Println("asdadas")
		var ac Loan
		if err := json.NewDecoder(r.Body).Decode(&ac); err != nil {
			return 0, err
		}
		fmt.Println(ac.UserID)
		idStr = strconv.Itoa(ac.UserID)
	} else {
		idStr = mux.Vars(r)["id"]
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return id, fmt.Errorf("invalid account id: %s", idStr)
	}
	return id, nil
}
