// Prepper is a web application for organising school prep labs and the
// activities they are preparing. It incorporates a scheduling system, booking
// system and inventory system in one package designed for ease of use by busy
// lab technicians.
//
// Written for A-Level computer science 2024 by Ethan Marshall.
// Copyright (C) 2023 - Ethan Marshall
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/ejv2/prepper/conf"
	"github.com/ejv2/prepper/data"
	"github.com/ejv2/prepper/isams"
	"github.com/ejv2/prepper/logging"
	"github.com/ejv2/prepper/maintenance"
	"github.com/ejv2/prepper/notifications"
	"github.com/ejv2/prepper/session"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	gormsql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Core application paths.
const (
	PathFrontend  = "frontend"
	PathTemplates = PathFrontend + string(os.PathSeparator) + "templates"
	PathStatic    = PathFrontend + string(os.PathSeparator) + "static"
	PathSample    = "config.sample.json"
	PathConfig    = "config.json"
)

// Lifetime application state.
var (
	Config        conf.Config
	Database      *gorm.DB
	Sessions      session.Store
	ISAMS         *isams.ISAMS
	Maintenance   maintenance.Manager
	MSched        maintenance.Scheduler
	Dmesg         *logging.Dmesg
	Notifications *notifications.Store
)

func loadConfig() error {
	c, err := conf.NewConfig(PathConfig)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("load config: %w", err)
		}

		log.Println("[WARNING]: config file not found; falling back to sample config")
		c, err = conf.NewConfig(PathSample)
		if err != nil {
			return fmt.Errorf("load fallback config: %w", err)
		}
	}

	Config = c
	return nil
}

func initDatabase(c conf.Config) (err error) {
	cfg := mysql.Config{
		Addr:                 c.Database.FullAddr(),
		DBName:               c.Database.Database,
		User:                 c.Database.Username,
		Passwd:               c.Database.Password,
		Net:                  "tcp",
		ParseTime:            true,
		Loc:                  time.UTC,
		AllowNativePasswords: true,
		RejectReadOnly:       true,
		Params: map[string]string{
			"charset": "utf8mb4",
		},
	}

	lvl := logger.Warn
	if c.DebugMode {
		lvl = logger.Info
	}

	lg := logger.New(log.Default(), logger.Config{
		LogLevel:                  lvl,
		Colorful:                  c.DebugMode,
		IgnoreRecordNotFoundError: !c.DebugMode,
	})

	Database, err = gorm.Open(gormsql.Open(cfg.FormatDSN()), &gorm.Config{Logger: lg})
	if err != nil {
		return err
	}

	return nil
}

func cleanBookings() error {
	n, err := data.CleanBookings(Database)
	if err != nil {
		return err
	}
	log.Println("Cleaned", n, "outdated bookings")
	return nil
}

func cleanDeleted() error {
	return data.CleanDeleted(Database)
}

func initRoutes(router *gin.Engine) {
	// Static assets path
	router.Static("/assets/", "frontend/static")

	// Site root
	router.GET("/", handleRoot)

	// About page
	router.GET("/about", handleAbout)

	// Help page
	router.GET("/help", session.Authenticator(&Sessions, true), handleHelp)

	// Login page
	router.GET("/login", handleLogin)
	router.POST("/login", handleLoginAttempt)

	// Logout page
	router.GET("/logout", handleLogout)

	// Dashboard (requires authentication)
	router.GET("/dashboard/", session.Authenticator(&Sessions, true), handleDashboard)

	// Account settings
	r := router.Group("/account/", session.Authenticator(&Sessions, true))
	{
		r.GET("/", handleAccounts)
		r.GET("/:id", handleEditAccount)
		r.GET("/:id/timetable", handleAccountTimetable)
		r.GET("/:id/unlink", handleAccountUnlink)
		r.GET("/:id/link", handleAccountLink)
		r.GET("/:id/sync", handleAccountSync)
		r.GET("/new", handleNewAccount)
		r.GET("/switch", handleAccountSwitch)
		r.GET("/password", handleChangePassword)
		r.POST("/password", handleChangePasswordAttempt)
	}

	// Technician todo list
	r = router.Group("/todo/", session.Permissions(&Sessions, Database, data.CapAllBooking, true))
	{
		r.GET("/", handleTodo)

		r.GET("/unread/:id", handleTodoUnread)
		r.GET("/progress/:id", handleTodoProgress)
		r.GET("/done/:id", handleTodoDone)
		r.GET("/reject/:id", handleTodoReject)
	}

	r = router.Group("/inventory/", session.Permissions(&Sessions, Database, data.CapManageInventory, true))
	{
		r.GET("/", handleInventory)
		r.GET("/item/:id", handleItem)
		r.GET("/item/:id/delete", handleItemDelete)
		r.GET("/item/:id/locate", handleItemLocate)
		r.GET("/new", handleNewItem)
		r.GET("/report", handleInventoryReport)
		r.GET("/locate", handleInventoryLocate)
	}

	r = router.Group("/activity/", session.Permissions(&Sessions, Database, data.CapManageInventory, true))
	{
		r.GET("/", handleActivities)
		r.GET("/new", handleActivityNew)
		r.GET("/:activity/edit", handleActivityEdit)
		r.POST("/:activity/edit", handleActivityDoEdit)
		r.GET("/:activity/delete", handleActivityDelete)
	}

	r = router.Group("/book/", session.Authenticator(&Sessions, true))
	{
		r.GET("/", handleBook)
		r.GET("/my", handleBookMy)

		r.GET("/:activity", handleBookActivity)
		r.GET("/:activity/timings", handleBookTimings)
		r.GET("/:activity/submit", handleBookSubmission)

		r.GET("/success/:id", handleBookSuccess)

		r.GET("/booking/:id", handleBooking)
		r.GET("/booking/:id/amend", handleBookAmend)
		r.POST("/booking/:id/amend", handleBookDoAmend)
		r.GET("/booking/:id/cancel", handleBookCancel)
	}

	r = router.Group("/api/")
	{
		r.Any("/", handleAPIRoot)
		r.POST("/user/edit/:id", handleAPIEditUser)
		r.GET("/dashboard", handleAPIDashboard)
		r.GET("/period", handleAPIPeriod)
		r.GET("/clashes", session.Authenticator(&Sessions, false), handleAPIClashes)

		r = r.Group("/item/", session.Permissions(&Sessions, Database, data.CapManageInventory, false))
		{
			r.POST("/create", handleAPICreateItem)
			r.Any("/edit", handleAPIBadEditItem)
			r.POST("/:id/edit", handleAPIEditItem)
		}
	}

	r = router.Group("/admin/", session.Permissions(&Sessions, Database, data.CapLogging, false))
	{
		r.Any("/", handleAdminRoot)
		r.GET("/logs", handleAdminLogs)
		r.GET("/error", handleAdminError)
		r.GET("/maintenance", handleAdminMaintenance)
		r.GET("/runnow", handleAdminRunMaint)
	}
}

func main() {
	// Setup global context
	ctx, cf := context.WithCancel(context.Background())
	defer cf()

	// Init early logging
	Dmesg = logging.NewDmesg()
	log.SetOutput(Dmesg.LogOutput())

	// Banner
	log.Print("Starting Prepper ", VersionString(), "...")

	// Parse config
	if err := loadConfig(); err != nil {
		log.Fatalln(err)
	}

	// ISAMS Support
	if Config.HasISAMS() {
		log.Println("Loading iSAMS data...")
		var err error
		ISAMS, err = isams.New(Config.ISAMS.Domain, Config.ISAMS.APIKey)
		if err != nil {
			log.Fatalln("iSAMS load:", err)
		}

		log.Print("ISAMS Support Enabled (connected to ", Config.ISAMS.Domain, ")")
	}

	// Init session storage
	Sessions = session.NewStore()

	// Notifications storage
	Notifications = notifications.NewStore()

	// Connect to database
	if err := initDatabase(Config); err != nil {
		log.Fatalln("database connection:", err)
	}
	if Config.DebugMode {
		// Migrate schema if needed
		log.Println("[WARNING]: Auto migrating database schema...")
		if Database.AutoMigrate(
			&data.User{},
			&data.Booking{}, &data.Activity{},
			&data.EquipmentSet{}, &data.EquipmentItem{},
		) != nil {
			log.Fatalln("Database migration failed")
		}
		log.Println("Auto migration complete")
	}
	log.Println("Connected to database on", Config.Database.FullAddr())

	// Setup gin debug mode
	if !Config.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// Startup and listen
	router := gin.New()
	srv := http.Server{
		Addr:              Config.FullAddr(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
	router.Use(gin.LoggerWithWriter(Dmesg.LogOutput()), gin.Recovery())
	gin.ForceConsoleColor()

	// Init maintenance manager
	mterr := make(chan error, 1)
	Maintenance = maintenance.NewManager(true)
	router.Use(maintenance.MiddlewareWithHandler(&Maintenance, func(c *gin.Context) {
		_, t := Maintenance.State()
		ts := "UNKNOWN"
		fend := time.Now()
		if t != nil {
			ts = t.Format(time.RFC1123)
			fend = t.Add(5 * time.Minute)
		}

		c.AbortWithStatus(http.StatusServiceUnavailable)
		fmt.Fprintln(c.Writer,
			"Prepper is currently down for maintenance and will be offline for 2-3 minutes.\n",
			"We apologise for any inconvenience.\n\n",
			"Start time:", ts, "\n",
			"Predicted End Time:", fend.Format(time.RFC1123),
		)
	}))
	MSched = maintenance.Scheduler{
		// TODO: add config variable for this
		Interval: &maintenance.StandardInterval,
		Manager:  &Maintenance,
		Handlers: []func() error{
			cleanBookings,
			cleanDeleted,
		},
		Ctx: ctx,
		Err: mterr,
	}
	go MSched.Run()
	go func() {
		for {
			log.Println("[MAINTENANCE ERROR]", <-mterr)
		}
	}()

	if err := router.SetTrustedProxies(Config.TrustedProxies); err != nil {
		log.Fatal("invalid proxy entries: ", err)
	}

	router.LoadHTMLGlob(PathTemplates + "/*")
	initRoutes(router)

	errchan := make(chan error, 1)
	sigchan := make(chan os.Signal, 1)

	signal.Notify(sigchan, os.Interrupt)
	log.Println("Server listening on", Config.FullAddr())
	go func() {
		err := srv.ListenAndServe()
		errchan <- err
	}()

	select {
	case <-sigchan:
		log.Println("Caught interrupt signal. Terminating gracefully...")
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			if err == ctx.Err() {
				log.Println("Shutdown timeout reached. Terminating forcefully...")
				return
			}
			log.Fatal(err)
		}
	case err := <-errchan:
		if err != http.ErrServerClosed {
			log.Panic(err)
		}
	}
}
