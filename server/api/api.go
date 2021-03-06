package api

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/valasek/timesheet/server/logger"
	"github.com/valasek/timesheet/server/models"
	"github.com/valasek/timesheet/server/version"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

// API -
type API struct {
	users           *models.UserManager
	consultants     *models.ConsultantManager
	reportedRecords *models.ReportedRecordManager
	projects        *models.ProjectManager
	rates           *models.RateManager
	holidays        *models.HolidayManager
}

// AppSettings -
type AppSettings struct {
	Version string `json:"version"`
	TimeZone string `json:"timeZone"`
	DailyWorkingHours float64 `json:"dailyWorkingHours"`
	DailyWorkingHoursMin float64 `json:"dailyWorkingHoursMin"`
	DailyWorkingHoursMax float64 `json:"dailyWorkingHoursMax"`
	Vacation string `json:"vacation"`
	YearlyVacationDays int64 `json:"yearlyVacationDays"`
	VacationPersonal string `json:"vacationPersonal"`
	YearlyPersonalDays int64 `json:"yearlyPersonalDays"`
	VacationSick string `json:"vacationSick"`
	YearlySickDays int64 `json:"yearlySickDays"`
	IsWorking string `json:"isWorking"`
	IsNonWorking string `json:"isNonWorking"`
}

// AppSettings returns list of all appliocation and user settings for configuration file
func (api *API) AppSettings(w http.ResponseWriter, req *http.Request) {
	settings := AppSettings{
		Version: version.Version,
		TimeZone: viper.GetString("timeZone"),
		DailyWorkingHours: viper.GetFloat64("dailyWorkingHours"),
		DailyWorkingHoursMin: viper.GetFloat64("dailyWorkingHoursMin"),
		DailyWorkingHoursMax: viper.GetFloat64("dailyWorkingHoursMax"),
		Vacation: viper.GetString("vacation"),
		YearlyVacationDays: viper.GetInt64("yearlyVacationDays"),
		VacationPersonal: viper.GetString("vacationPersonal"),
		YearlyPersonalDays: viper.GetInt64("yearlyPersonalDays"),
		VacationSick: viper.GetString("vacationSick"),
		YearlySickDays: viper.GetInt64("yearlySickDays"),
		IsWorking: viper.GetString("isWorking"),
		IsNonWorking: viper.GetString("isNonWorking"),
	}
	json.NewEncoder(w).Encode(settings)
}

// Download -
func (api *API) Download(w http.ResponseWriter, req *http.Request) {
	fileName, err := export()
	if err != nil {
		http.Error(w, "downloading data failed with error: "+err.Error(), 404)
		return
	}

	file, err := os.Open(fileName)
	defer file.Close()
	if err != nil {
		http.Error(w, "File not found.", 404)
		return
	}

	//Get the Content-Type of the file
	//Create a buffer to store the header of the file in
	FileHeader := make([]byte, 512)
	//Copy the headers into the FileHeader buffer
	file.Read(FileHeader)
	//Get content type of file
	FileContentType := http.DetectContentType(FileHeader)

	//Get the file size
	FileStat, _ := file.Stat()                         //Get info from file
	FileSize := strconv.FormatInt(FileStat.Size(), 10) //Get file size as a string

	//Send the headers
	w.Header().Set("Content-Disposition", "attachment; filename=go.mod")
	w.Header().Set("Content-Type", FileContentType)
	w.Header().Set("Content-Length", FileSize)

	//Send the file
	//We read 512 bytes from the file already, so we reset the offset back to 0
	file.Seek(0, 0)
	io.Copy(w, file)
	return
}

// DownloadLogs -
func (api *API) DownloadLogs(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	logLevel := vars["logLevel"]
	if len(logLevel) < 1 {
		logger.Log.Error("DownloadLogs, param 'logLevel' is missing")
		return
	}

	file := ""
	switch logLevel {
	case "0":
		file = "info.log"
	case "1":
		file = "error.log"
	default:
		logger.Log.Error("DownloadLogs, unknown logLevel ", logLevel)
	}
	fileName := path.Join(viper.GetString("logFolder"), file)
	f, err := os.Open(fileName)
	defer f.Close()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, file+" contains no log entries")
		return
	}

	http.ServeFile(w, req, fileName)
}

// NewAPI -
func NewAPI(db *models.DB) *API {

	// usermgr, _ := models.NewUserManager(db)
	consultantmgr, err := models.NewConsultantManager(db)
	if err != nil {
		logger.Log.Error(err)
	}
	reportedrecordsmgr, err := models.NewReportedRecordManager(db)
	if err != nil {
		logger.Log.Error(err)
	}
	projectsmgr, err := models.NewProjectManager(db)
	if err != nil {
		logger.Log.Error(err)
	}
	ratesmgr, err := models.NewRateManager(db)
	if err != nil {
		logger.Log.Error(err)
	}
	holidaysmgr, err := models.NewHolidayManager(db)
	if err != nil {
		logger.Log.Error(err)
	}

	return &API{
		// users:  usermgr,
		consultants:     consultantmgr,
		reportedRecords: reportedrecordsmgr,
		projects:        projectsmgr,
		rates:           ratesmgr,
		holidays:        holidaysmgr,
	}
}

// ResetAPI - drops and creates all empty tables
func ResetAPI(db *models.DB) {

	// db.DropTableIfExists(&models.Users{})
	db.DropTableIfExists(&models.Consultant{})
	db.DropTableIfExists(&models.ReportedRecord{})
	db.DropTableIfExists(&models.Rate{})
	db.DropTableIfExists(&models.Project{})
	db.DropTableIfExists(&models.Holiday{})

	logger.Log.Info("recreated tables:")
	// usermgr, err := models.NewUserManager(db)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println("- users")
	_, err := models.NewConsultantManager(db)
	if err != nil {
		logger.Log.Error("recreated tables:", err)
	}
	logger.Log.Info("- consultants")
	_, err = models.NewReportedRecordManager(db)
	if err != nil {
		logger.Log.Error("recreated tables:", err)
	}
	logger.Log.Info("- reported_records")
	models.NewProjectManager(db)
	if err != nil {
		logger.Log.Error("recreated tables:", err)
	}
	logger.Log.Info("- projects")
	models.NewRateManager(db)
	if err != nil {
		logger.Log.Error("recreated tables:", err)
	}
	logger.Log.Info("- rates")
	models.NewHolidayManager(db)
	if err != nil {
		logger.Log.Error("recreated tables:", err)
	}
	logger.Log.Info("- holidays")
}

// SeedAPI - loads initial data into DB
func SeedAPI(db *models.DB, table string) {
	api := NewAPI(db)
	logger.Log.Info("Loaded table, # of records, filename:")
	switch table {
	case "rates", "consultants", "projects", "reported_records", "holidays":
		SeedTable(api, table)
	case "all":
		// users
		SeedTable(api, "rates")
		SeedTable(api, "consultants")
		SeedTable(api, "projects")
		SeedTable(api, "reported_records")
		SeedTable(api, "holidays")
	default:
		logger.Log.Error("unable to seed non-existent table: ", table)
	}
}

// SeedTable -
func SeedTable(api *API, table string) (count int) {
	switch table {
	case "rates":
		if api.rates.RateCount() > 0 {
			logger.Log.Warn(fmt.Sprintf("- rates, file %s skipped, table contains %d records", viper.GetString("data.rates"), api.rates.RateCount()))
			return 0
		}
		count = api.rates.RateSeed("./data/" + viper.GetString("data.rates"))
		logger.Log.Info(fmt.Sprintf("- rates, %d records, %s", count, viper.GetString("data.rates")))
	case "consultants":
		if api.consultants.ConsultantCount() > 0 {
			logger.Log.Warn(fmt.Sprintf("- consultants, file %s skipped, table contains %d records", viper.GetString("data.consultants"), api.consultants.ConsultantCount()))
			return 0
		}
		count = api.consultants.ConsultantSeed("./data/" + viper.GetString("data.consultants"))
		logger.Log.Info(fmt.Sprintf("- consultants, %d records, %s", count, viper.GetString("data.consultants")))
	case "projects":
		if api.projects.ProjectCount() > 0 {
			logger.Log.Warn(fmt.Sprintf("- projects, file %s skipped, table contains %d records", viper.GetString("data.projects"), api.projects.ProjectCount()))
			return 0
		}
		count = api.projects.ProjectSeed("./data/" + viper.GetString("data.projects"))
		logger.Log.Info(fmt.Sprintf("- projects, %d records, %s", count, viper.GetString("data.projects")))
	case "reported_records":
		if api.reportedRecords.ReportedRecordCount() > 0 {
			logger.Log.Warn(fmt.Sprintf("- reported_records, file %s skipped, table contains %d records", viper.GetString("data.reportedRecords"), api.reportedRecords.ReportedRecordCount()))
			return 0
		}
		count = api.reportedRecords.ReportedRecordSeed("./data/" + viper.GetString("data.reportedRecords"))
		logger.Log.Info(fmt.Sprintf("- reported_records, %d records, %s", count, viper.GetString("data.reportedRecords")))
	case "holidays":
		if api.holidays.HolidayCount() > 0 {
			logger.Log.Warn(fmt.Sprintf("- holidays, file %s skipped, table contains %d records", viper.GetString("data.holidays"), api.holidays.HolidayCount()))
			return 0
		}
		count = api.holidays.HolidaySeed("./data/" + viper.GetString("data.holidays"))
		logger.Log.Info(fmt.Sprintf("- holidays, %d records, %s", count, viper.GetString("data.holidays")))
	default:
		logger.Log.Warn("unknown table to seed: ", table)
	}
	return count
}

// CheckAndInitAPI - loads initial data into DB
func CheckAndInitAPI(db *models.DB) (api *API) {
	logger.Log.Info("checking DB ...")
	emptyTable := false
	api = NewAPI(db)
	if api.rates.RateCount() == 0 {
		SeedTable(api, "rates")
		emptyTable = true
	}
	if api.consultants.ConsultantCount() == 0 {
		SeedTable(api, "consultants")
		emptyTable = true
	}
	if api.projects.ProjectCount() == 0 {
		SeedTable(api, "projects")
		emptyTable = true
	}
	if api.reportedRecords.ReportedRecordCount() == 0 {
		SeedTable(api, "reported_records")
		emptyTable = true
	}
	if api.holidays.HolidayCount() == 0 {
		SeedTable(api, "holidays")
		emptyTable = true
	}
	if emptyTable {
		logger.Log.Info("loaded missing required data (see tables above)")
	}
	return api
}

// BackupAPI - drops and creates all empty tables
func BackupAPI(rotation int, folder string, db *models.DB) {

	api := NewAPI(db)
	id := time.Now().Format("2006-01-02_150405")

	logger.Log.Info("backuped tables:")
	tableNames := []string{"rates", "projects", "reported_records", "consultants", "holidays"}
	for _, baseFileName := range tableNames {
		err := rotateBackupFile(rotation, folder, baseFileName)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("not able to rotate %s backup files, backups stopped, handle the error: %s", baseFileName, err))
		}

		fileName := baseFileName + "_" + id + ".csv"
		filePath := filepath.Join(folder, fileName)
		n := 0
		switch baseFileName {
		case "projects":
			n, err = api.projects.ProjectBackup(filePath)
		case "rates":
			n, err = api.rates.RateBackup(filePath)
		case "consultants":
			n, err = api.consultants.ConsultantBackup(filePath)
		case "holidays":
			n, err = api.holidays.HolidayBackup(filePath)
		case "reported_records":
			n, err = api.reportedRecords.ReportedRecordBackup(filePath)
		}
		if err != nil {
			logger.Log.Error(fmt.Sprintf("backuped tables: error during %s backup: %s", baseFileName, err))
		} else {
			logger.Log.Info(fmt.Sprintf("- %s, %d records", baseFileName, n))
		}
	}
}

// ConnectDB connects and pings DB
func ConnectDB() (db *models.DB) {
	switch DBType := viper.GetString("dbType"); DBType {
	case "postgresql":
		// DBhost, DBport, DBuser, DBpassword, DBname, SSLmode, url, port := "", "", "", "", "", "", "", ""
		dbURL := viper.GetString("DATABASE_URL")

		if len(dbURL) == 0 {
			dbURL = "host=" + viper.GetString("postgresql.host") +
			" port=" + viper.GetString("postgresql.port") +
			" user=" + viper.GetString("postgresql.user") +
			" dbname=" + viper.GetString("postgresql.name") +
			" password=" + viper.GetString("postgresql.password") +
			" sslmode=" + viper.GetString("postgresql.SSLMode")
		}
		logger.Log.Info("connecting to DB ", dbURL)
		db = models.NewPostgresDB(dbURL)
		// fmt.Println("connected to DB:  ", connectionString)
		logger.Log.Info("connected to DB ", dbURL)
		// fmt.Println("")
	default:
		logger.Log.Error("not able to connect to DB, supported DB types (postgresql), set: ", DBType)
		os.Exit(1)
	}
	return db
}

func rotateBackupFile(rotation int, folder, baseFileName string) error {
	files, err := ioutil.ReadDir(filepath.Clean(folder))
	if err != nil {
		return err
	}

	oldestTime := time.Now()
	var oldestFile os.FileInfo
	var filteredNames []string
	if len(files) == 0 {
		return nil
	}
	for _, file := range files {
		if strings.Contains(file.Name(), baseFileName) {
			filteredNames = append(filteredNames, file.Name())
			if file.Mode().IsRegular() && file.ModTime().Before(oldestTime) {
				oldestFile = file
				oldestTime = file.ModTime()
			}
		}
	}
	if len(filteredNames) >= rotation {
		err := os.Remove(filepath.Join(folder, oldestFile.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}

func appendFiles(filename string, zipw *zip.Writer) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open %s: %s", filename, err)
	}
	defer file.Close()

	wr, err := zipw.Create(filename)
	if err != nil {
		msg := "failed to create entry for %s in zip file: %s"
		return fmt.Errorf(msg, filename, err)
	}

	if _, err := io.Copy(wr, file); err != nil {
		return fmt.Errorf("failed to write %s to zip: %s", filename, err)
	}

	return nil
}

func cleanExportedFiles(folder string) error {
	dir, err := os.Open(folder)
	if err != nil {
		return err
	}
	files, err := dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, f := range files {
		fName := f.Name()
		fNamePath := filepath.Join(folder, fName)
		os.Remove(fNamePath)
	}
	return nil
}

// exports all data from DB into file timesheet-backup.zip
func export() (fileName string, err error) {
	fileName = "timesheet-backup.zip"
	db := ConnectDB()
	defer db.Close()
	exportFolder := viper.GetString("export.location")

	err = cleanExportedFiles(exportFolder)
	if err != nil {
		return "", err
	}

	BackupAPI(viper.GetInt("backup.rotation"), exportFolder, db)
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	file, err := os.OpenFile("timesheet-backup.zip", flags, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	files, err := ioutil.ReadDir(exportFolder)
	if err != nil {
		return "", err
	}

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	for _, file := range files {
		err := appendFiles(filepath.Join(exportFolder, file.Name()), zipw)
		if err != nil {
			return "", err
		}
	}

	return fileName, nil
}
