package main

import (
	"bufio"
	"dogukan-dev/tuition/db"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TransactionStatus provides a general status for operations like adding or paying tuition.
type TransactionStatus struct {
	Status  string `json:"status"`            // e.g., "Successful", "Error", "Pending"
	Message string `json:"message,omitempty"` // Detailed message about the result.
}

func uploadCSVHandler(w http.ResponseWriter, r *http.Request) {
}

// Handlers

// Mobile /Banking - Query Tuition (No Auth, No Paging)
func (a *App) QueryTuitionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	studentNo := r.URL.Query().Get("student_no")
	activeTerm := r.URL.Query().Get("active_term")
	if studentNo == "" || activeTerm == "" {
		http.Error(w, `{"error":"student_no and active_term parameters are required"}`, http.StatusBadRequest)
		return
	}

	student, err := a.Queries.GetStudentById(a.Context, studentNo)
	if err != nil {

		http.Error(w, `{"error":"Student not found"}`, http.StatusNotFound)
		return
	}

	signedInStudent := r.Context().Value("LOGGEDIN_STUDENT_NO")

	if signedInStudent == "" {
		http.Error(w, `{"error":"Please don't delete your cookies.Try logout and login again"}`, http.StatusNotFound)
		return
	}

	if signedInStudent != student.StudentNo {
		http.Error(w, `{"error":"Each student only can see their own tuitions"}`, http.StatusNotFound)
		return
	}

	term, err := a.Queries.GetTuitionByTerm(a.Context, db.GetTuitionByTermParams{
		StudentNo: studentNo,
		Term:      activeTerm,
	})

	if err != nil {
		http.Error(w, `{"error":"Cannot query term"}`, http.StatusBadRequest)
		return
	}

	if len(term) == 0 {
		http.Error(w, `{"error":"There is no added tuition for you this term"}`, http.StatusBadRequest)
		return
	}

	type TuitionQueryResponse struct {
		StudentNo    string
		Term         string
		TuitionTotal float64
		Balance      float64
	}

	response := TuitionQueryResponse{
		StudentNo:    student.StudentNo,
		TuitionTotal: term[0].TuitionTotal,
		Term:         term[0].Term,
		Balance:      student.Balance,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Banking App - Pay Tuition (No Auth, No Paging)
func (a *App) PayTuitionHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: rate limit on gateway level
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type PaymentRequest struct {
		StudentNo string
		Term      string
		Amount    float64
	}

	var req PaymentRequest
	q := r.URL.Query()
	req.StudentNo = q.Get("student_no")
	req.Term = q.Get("term")

	// Convert amount from string to float64
	amountStr := q.Get("amount")
	if amountStr != "" {
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid amount"}`, http.StatusBadRequest)
			return
		}
		req.Amount = amount
	}

	if req.StudentNo == "" || req.Term == "" {
		http.Error(w, `{"error":"student_no and term are required"}`, http.StatusBadRequest)
		return
	}

	type PaymentResponse struct {
		TransactionStatus
		Balance float64 `json:"balance,omitempty"`
	}

	student, err := a.Queries.GetStudentById(a.Context, req.StudentNo)

	if err != nil {
		response := PaymentResponse{
			TransactionStatus: TransactionStatus{
				Status:  "Error",
				Message: "Student with this number does not exist",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.Amount <= 0 {
		response := PaymentResponse{
			TransactionStatus: TransactionStatus{
				Status:  "Error",
				Message: "You must enter an amount first",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	term, err := a.Queries.GetTuitionByTerm(a.Context, db.GetTuitionByTermParams{
		StudentNo: req.StudentNo,
		Term:      req.Term,
	})
	if err != nil {
		http.Error(w, `{"error":"Cannot get term info"}`, http.StatusBadRequest)
		return
	}

	if len(term) == 0 {
		http.Error(w, `{"error":"There is no tuition set for this term"}`, http.StatusBadRequest)
		return
	}

	balanceSum := student.Balance + req.Amount

	if balanceSum < student.TuitionTotal.Float64 {

		a.Queries.UpdateBalance(a.Context, db.UpdateBalanceParams{
			StudentNo: student.StudentNo,
			Balance:   balanceSum,
		})

		response := PaymentResponse{
			TransactionStatus: TransactionStatus{
				Status:  "Successful",
				Message: fmt.Sprintf("Entered amount added to balance.Balance: %.2f", balanceSum),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	if balanceSum >= student.TuitionTotal.Float64 {
		currentBalance := balanceSum - student.TuitionTotal.Float64

		a.Queries.UpdateBalance(a.Context, db.UpdateBalanceParams{
			StudentNo: student.StudentNo,
			Balance:   currentBalance,
		})
		a.Queries.ResetTuitionTotal(a.Context, db.ResetTuitionTotalParams{
			StudentNo: student.StudentNo,
			Term:      req.Term,
		})

		response := PaymentResponse{
			TransactionStatus: TransactionStatus{
				Status:  "Successful",
				Message: fmt.Sprintf("You paid this term's tuition.Any excess amount added to balance.\n Balance: %.2f", currentBalance),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

}

// User - Register To system
func (a *App) registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type RegisterRequest struct {
		StudentNo   string `json:"student_no"`
		RawPassword string `json:"password"`
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.StudentNo == "" || req.RawPassword == "" {
		http.Error(w, `{"error":"student_no and hashed_password are required"}`, http.StatusBadRequest)
		return
	}

	hashedPassword, err := HashPassword(req.RawPassword)
	if err != nil {
		http.Error(w, `{"error":"cannot hash password"}`, http.StatusBadRequest)
		return
	}
	err = a.Queries.AddStudentAccount(a.Context, db.AddStudentAccountParams{
		StudentNo:      req.StudentNo,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		http.Error(w, `{"error":"Student Cannot be add to system.It might be already registered"}`, http.StatusBadRequest)
		return
	}

	jwt, err := GenerateJWT(req.StudentNo)
	setJWTCookie(w, jwt)

	response := TransactionStatus{
		Status:  "Success",
		Message: fmt.Sprintf("You've successfully registered to system.\nToken: %s", jwt),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// User - Login Into system
func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type LoginRequest struct {
		StudentNo   string `json:"student_no"`
		RawPassword string `json:"password"`
	}
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.StudentNo == "" || req.RawPassword == "" {
		http.Error(w, `{"error":"student_no and password are required"}`, http.StatusBadRequest)
		return
	}
	account, err := a.Queries.GetAccountByStudentNo(a.Context, req.StudentNo)
	if err != nil {
		http.Error(w, `{"error":"Cannot get account by student No"}`, http.StatusBadRequest)
		return

	}
	err = bcrypt.CompareHashAndPassword([]byte(account.HashedPassword), []byte(req.RawPassword))

	if err != nil {
		http.Error(w, `{"error":"Wrong Password"}`, http.StatusBadRequest)
		return
	}

	jwt, err := GenerateJWT(req.StudentNo)
	setJWTCookie(w, jwt)

	response := TransactionStatus{
		Status:  "Success",
		Message: fmt.Sprintf("You've successfully logged into system.\nToken: %s", jwt),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Admin - Add Student To System
func (a *App) addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type AddStudentRequest struct {
		StudentNo string
		Balance   float64
	}
	var req AddStudentRequest
	q := r.URL.Query()
	req.StudentNo = q.Get("student_no")

	// Convert amount from string to float64
	balaceStr := q.Get("balance")
	if balaceStr != "" {
		balance, err := strconv.ParseFloat(balaceStr, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid limit"}`, http.StatusBadRequest)
			return
		}
		req.Balance = balance
	}

	if req.StudentNo == "" || req.Balance <= 1 {
		http.Error(w, `{"error":"student_no and balance(must be at least 1) are required"}`, http.StatusBadRequest)
		return
	}

	err := a.Queries.AddNewStudent(a.Context, db.AddNewStudentParams{
		StudentNo: req.StudentNo,
		Balance:   req.Balance,
	})
	if err != nil {
		http.Error(w, `{"error":"Student Cannot be add to system"}`, http.StatusBadRequest)
		return
	}

	response := TransactionStatus{
		Status:  "Success",
		Message: fmt.Sprintf("Student %s with balance of %.2f added to tuition system", req.StudentNo, req.Balance),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Admin - Add Tuition (Single)
func (a *App) addTuitionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type AddTuitionRequest struct {
		StudentNo     string
		Term          string
		TuitionAmount float64
	}
	var req AddTuitionRequest
	req.StudentNo = r.URL.Query().Get("student_no")
	req.Term = r.URL.Query().Get("term")
	// Convert amount from string to float64
	amountStr := r.URL.Query().Get("tuition_amount")
	if amountStr != "" {
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid amount"}`, http.StatusBadRequest)
			return
		}
		req.TuitionAmount = amount
	}

	if req.StudentNo == "" || req.Term == "" || req.TuitionAmount <= 1 {
		http.Error(w, `{"error":"student_no, term, and valid tuition amount are required"}`, http.StatusBadRequest)
		return
	}

	student, err := a.Queries.GetStudentById(a.Context, req.StudentNo)
	if err != nil {
		http.Error(w, `{"error":"There is no student with this number"}`, http.StatusBadRequest)
		return
	}

	term, err := a.Queries.GetTuitionByTerm(a.Context, db.GetTuitionByTermParams{
		StudentNo: req.StudentNo,
		Term:      req.Term,
	})
	if err != nil {
		http.Error(w, `{"error":"Cannot get term info"}`, http.StatusBadRequest)
		return
	}

	if len(term) > 0 { // If entered term is set already
		response := TransactionStatus{
			Status:  "Error",
			Message: fmt.Sprintf("This student's tuition for this term is already set"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	err = a.Queries.AddTuitionToOneStudent(a.Context, db.AddTuitionToOneStudentParams{
		StudentNo:    student.StudentNo,
		Term:         req.Term,
		TuitionTotal: req.TuitionAmount,
	})

	response := TransactionStatus{
		Status:  "Success",
		Message: fmt.Sprintf("Tuition of %.2f added for student %s, term %s  ", req.TuitionAmount, req.StudentNo, req.Term),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}

// Admin - Add Tuition (Multiple)
func (a *App) addTuitionBatchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	type Tuition struct {
		StudentNo     string
		Term          string
		TuitionAmount float64
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error reading file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// // Save file locally (optional)
	// out, err := os.Create("./uploads/" + header.Filename)
	// if err != nil {
	// 	http.Error(w, "Cannot create file: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// defer out.Close()
	//
	// _, err = io.Copy(out, file)
	// if err != nil {
	// 	http.Error(w, "Cannot save file: "+err.Error(), http.StatusInternalServerError)
	// 	return
	// }

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	if err != nil {
		http.Error(w, `cannot read csv file`, http.StatusBadRequest)
		panic(err)
	}
	var tuitions []Tuition

	// skip header
	for _, row := range records[1:] {
		amt, _ := strconv.ParseFloat(row[2], 64)

		tuitions = append(tuitions, Tuition{
			StudentNo:     row[0],
			Term:          row[1],
			TuitionAmount: amt,
		})
	}

	for _, t := range tuitions {
		student, err := a.Queries.GetStudentById(a.Context, t.StudentNo)
		if err != nil {
			http.Error(w, `{"error":"There is no student with this number"}`, http.StatusBadRequest)
			return
		}

		term, err := a.Queries.GetTuitionByTerm(a.Context, db.GetTuitionByTermParams{
			StudentNo: t.StudentNo,
			Term:      t.Term,
		})
		if err != nil {
			http.Error(w, `{"error":"Cannot get term info"}`, http.StatusBadRequest)
			return
		}

		if len(term) > 0 { // If entered term is set already
			response := TransactionStatus{
				Status:  "Error",
				Message: fmt.Sprintf("This student's tuition for this term is already set"),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		err = a.Queries.AddTuitionToOneStudent(a.Context, db.AddTuitionToOneStudentParams{
			StudentNo:    student.StudentNo,
			Term:         t.Term,
			TuitionTotal: t.TuitionAmount,
		})

		response := TransactionStatus{
			Status:  "Success",
			Message: fmt.Sprintf("Tuition of %.2f added for student %s, term %s  ", t.TuitionAmount, t.StudentNo, t.Term),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	}

}

// Admin - Unpaid Tuition Status
func (a *App) getLogsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var response []string

	file, err := os.Open("logs/api_requests.log")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		response = append(response, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Scanner error:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Admin - Unpaid Tuition Status
func (a *App) unpaidTuitionStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	limitInt := 10
	offsetInt := 0

	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	if limit != "" {
		tmp, err := strconv.Atoi(limit)
		if err != nil {
			http.Error(w, `Limit must be a number`, http.StatusBadRequest)
		}
		limitInt = tmp
	}
	if offset != "" {
		tmp, err := strconv.Atoi(offset)
		if err != nil {
			http.Error(w, `Offset must be a number`, http.StatusBadRequest)
		}

		offsetInt = tmp
	}

	type UnpaidStudent struct {
		StudentNumber string
		Term          string
	}
	var response []UnpaidStudent

	unpaid, err := a.Queries.UnpaidTuitions(a.Context, db.UnpaidTuitionsParams{
		Limit:  int32(limitInt),
		Offset: int32(offsetInt),
	})
	if err != nil {
		http.Error(w, `Unpaid Tuitions cannot be queried`, http.StatusBadRequest)
		panic(err)
	}

	for idx, _ := range unpaid {
		response = append(response, UnpaidStudent{
			StudentNumber: unpaid[idx].StudentNo,
			Term:          unpaid[idx].Term,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Health check
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}
