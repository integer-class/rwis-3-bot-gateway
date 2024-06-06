package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"strings"
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

type HouseholdData struct {
	NumberOfKK  string
	Address     string
	NumberOfRT  string
	NumberOfRW  string
	SubDistrict string
	City        string
	Province    string
	PostalCode  string
}

func handlePersonalDataRequest(ctx context.Context, db *pgxpool.Pool, sender string, include string) (error, string) {
	if include == "personal" {
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
	if include == "household" {
		log.Debug().Msgf("Handling household request from %s", sender)
		// handle household data request
		row := db.QueryRow(ctx, `
SELECT
	number_kk,
	address,
	rt,
	rw,
	sub_district,
	city,
	province,
	postal_code
FROM household
JOIN resident r on household.resident_id = r.resident_id
WHERE r.whatsapp_number = $1`, sender)
		var householdData HouseholdData
		err := row.Scan(
			&householdData.NumberOfKK,
			&householdData.Address,
			&householdData.NumberOfRT,
			&householdData.NumberOfRW,
			&householdData.SubDistrict,
			&householdData.City,
			&householdData.Province,
			&householdData.PostalCode,
		)
		if err != nil {
			return errors.Wrap(err, "failed to get household data"), ""
		}

		return nil, fmt.Sprintf(`*Data rumah tangga Anda*:
*Nomor KK*: %s
*Alamat*: %s
*RT*: %s
*RW*: %s
*Kelurahan*: %s
*Kota*: %s
*Provinsi*: %s
*Kode Pos*: %s`,
			householdData.NumberOfKK,
			householdData.Address,
			householdData.NumberOfRT,
			householdData.NumberOfRW,
			householdData.SubDistrict,
			householdData.City,
			householdData.Province,
			householdData.PostalCode)
	}
	if include == "household_all" {
		log.Debug().Msgf("Handling household members request from %s", sender)
		// handle household all members data request
		rows, err := db.Query(ctx, `
SELECT nik,
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
WHERE household_id IN (SELECT household_id
                       FROM resident
                       WHERE whatsapp_number = $1)`, sender)
		if err != nil {
			return errors.Wrap(err, "failed to get household all members data"), ""
		}

		var householdMembersData []PersonalData
		for rows.Next() {
			var personalData PersonalData
			err := rows.Scan(
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
				return errors.Wrap(err, "failed to scan household all members data"), ""
			}

			householdMembersData = append(householdMembersData, personalData)
		}

		var sb strings.Builder
		for _, member := range householdMembersData {
			sb.WriteString(fmt.Sprintf(`*Data kependudukan*:
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
*Nomor WhatsApp*: %s

`,
				member.Nik,
				member.FullName,
				member.PlaceOfBirth,
				member.DateOfBirth.Format("02 January 2006"),
				member.Gender,
				member.BloodType,
				member.Religion,
				member.MarriageStatus,
				member.Nationality,
				member.RangeIncome,
				member.Job,
				member.WhatsappNumber,
			))
		}
		return nil, sb.String()
	}
	return errors.New("invalid include type"), ""
}

func handleIssueReport(ctx context.Context, db *pgxpool.Pool, sender string, reply string, title string, description string) (error, string) {
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
	_, err = trx.Exec(ctx, `INSERT INTO issue_report (resident_id, title, description, status, approval_status)
VALUES ($1, $2, $3, $4, $5)`, residentId, title, description, "To do", "Pending")

	if err != nil {
		return errors.Wrap(err, "failed to insert issue report"), ""
	}

	err = trx.Commit(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to commit issue report transaction"), ""
	}

	if reply == "" {
		reply = "Terima kasih atas laporan Anda. Kami akan segera menindaklanjuti."
	}

	return nil, reply
}
