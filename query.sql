-- name: GetAccountByStudentNo :one
SELECT * FROM account
WHERE student_no = $1;

-- name: GetStudentById :one
SELECT * FROM student
LEFT JOIN tuition
ON student.student_no = tuition.student_no
WHERE student.student_no = $1;

-- name: GetStudentDailyLimit :one
SELECT daily_payment_limit 
FROM student
WHERE student_no = $1;

-- name: GetTuitionByTerm :many
SELECT * FROM student
INNER JOIN tuition
ON student.student_no = tuition.student_no
WHERE student.student_no = $1
AND tuition.term = $2;

-- name: AddNewStudent :exec
INSERT INTO student(student_no,balance)
VALUES ($1,$2)
RETURNING student_no;

-- name: AddStudentAccount :exec
INSERT INTO account(student_no,hashed_password)
VALUES ($1,$2);

-- name: UpdateBalance :exec
UPDATE student
SET balance = $2
WHERE student_no = $1;

-- name: DecreasePaymentLimit :exec
UPDATE student
SET daily_payment_limit = daily_payment_limit-1
WHERE student_no = $1;

-- name: ResetTuitionTotal :exec
UPDATE tuition
SET tuition_total = 0
WHERE student_no = $1 
AND term = $2;

-- name: AddTuitionToOneStudent :exec
INSERT INTO tuition(student_no,term,tuition_total)
VALUES ($1,$2,$3);

-- name: UnpaidTuitions :many
SELECT *
FROM student
INNER JOIN tuition
ON student.student_no = tuition.student_no
WHERE tuition.tuition_total > 0
LIMIT $1 OFFSET $2;
