CREATE TABLE IF NOT EXISTS account (
    account_no                  SERIAL PRIMARY KEY,
    student_no                  VARCHAR(11) NOT NULL UNIQUE,
    hashed_password             VARCHAR(148) NOT NULL,

    CONSTRAINT fk_student FOREIGN KEY (student_no) REFERENCES student(student_no)
);

CREATE TABLE IF NOT EXISTS student (
    student_no          VARCHAR(11) PRIMARY KEY,
    balance             DOUBLE PRECISION NOT NULL,
    daily_payment_limit INT NOT NULL DEFAULT 3,
    CONSTRAINT daily_payment_limit_nonnegative CHECK (daily_payment_limit >= 0)
);

CREATE TABLE IF NOT EXISTS tuition (
    tuition_id          SERIAL PRIMARY KEY,
    student_no          VARCHAR(11) NOT NULL,
    term                VARCHAR(50) NOT NULL,
    tuition_total       DOUBLE PRECISION NOT NULL,

    CONSTRAINT fk_student FOREIGN KEY (student_no) REFERENCES student(student_no)
);
