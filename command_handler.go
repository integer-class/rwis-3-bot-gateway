package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"time"
)

type PersonalData struct {
	Nik            string
	Name           string
	FullName       string
	PlaceOfBirth   string
	DateOfBirth    time.Time
	Gender         string
	BloodType      string
	Religion       string
	MarriageStatus string
	Nationality    string
	RangeIncome    string
	Job            string
	WhatsappNumber string
}

func handlePersonalDataRequest(ctx context.Context, db *pgxpool.Pool, sender string) (error, string) {
	log.Debug().Msgf("Handling personal data request from %s", sender)
	// handle personal data request
	row := db.QueryRow(ctx, `
SELECT 
    nik,
    full_name,
    place_of_birth,
    date_of_birth,
    gender,
    blood_type,
    religion,
    marriage_status,
    nationality,
    range_income,
    job,
    whatsapp_number
FROM resident 
WHERE whatsapp_number = $1`, sender)
	var personalData PersonalData
	err := row.Scan(
		&personalData.Nik,
		&personalData.FullName,
		&personalData.PlaceOfBirth,
		&personalData.DateOfBirth,
		&personalData.Gender,
		&personalData.BloodType,
		&personalData.Religion,
		&personalData.MarriageStatus,
		&personalData.Nationality,
		&personalData.RangeIncome,
		&personalData.Job,
		&personalData.WhatsappNumber,
	)
	if err != nil {
		return errors.Wrap(err, "failed to get personal data"), ""
	}

	return nil, fmt.Sprintf(`*Data kependudukan Anda*:
*NIK*: %s
*Nama Lengkap*: %s
*Tempat Lahir*: %s
*Tanggal Lahir*: %s
*Jenis Kelamin*: %s
*Golongan Darah*: %s
*Agama*: %s
*Status Pernikahan*: %s
*Kewarganegaraan*: %s
*Rentang Penghasilan*: %s
*Pekerjaan*: %s
*Nomor WhatsApp*: %s`,
		personalData.Nik,
		personalData.FullName,
		personalData.PlaceOfBirth,
		personalData.DateOfBirth.Format("02 January 2006"),
		personalData.Gender,
		personalData.BloodType,
		personalData.Religion,
		personalData.MarriageStatus,
		personalData.Nationality,
		personalData.RangeIncome,
		personalData.Job,
		personalData.WhatsappNumber)
}

func handleIssueReport(ctx context.Context, db *pgxpool.Pool, sender string, title string, description string) (error, string) {
	log.Debug().Msgf("Handling issue report from %s", sender)
	trx, err := db.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction"), ""
	}
	defer func(trx pgx.Tx, ctx context.Context) {
		err := trx.Rollback(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to rollback transaction")
		}
	}(trx, ctx)

	resident := trx.QueryRow(ctx, `
SELECT resident_id
FROM resident
WHERE whatsapp_number = $1`, sender)

	var residentId int
	err = resident.Scan(&residentId)
	if err != nil {
		return errors.Wrap(err, "failed to scan resident"), ""
	}

	// handle issue report
	_, err = trx.Exec(ctx, `
INSERT INTO issue_report (resident_id, title, description, status)
VALUES ($1, $2, $3, $4)`, residentId, title, description, "todo")

	if err != nil {
		return errors.Wrap(err, "failed to insert issue report"), ""
	}

	err = trx.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to commit issue report transaction"), ""
	}

	return nil, "Terima kasih atas laporan Anda. Kami akan segera menindaklanjuti."
}
