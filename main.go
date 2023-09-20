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

var (
	Config   conf.Config
	Database *gorm.DB
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

func main() {
	// Banner
	log.Print("Starting Prepper ", VersionString(), "...")

	// Parse config
	if err := loadConfig(); err != nil {
		log.Fatalln(err)
	}

	// Connect to database
	if err := initDatabase(Config); err != nil {
		log.Fatalln("database connection:", err)
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
	router.Use(gin.Logger(), gin.Recovery())

	if err := router.SetTrustedProxies(Config.TrustedProxies); err != nil {
		log.Fatal("invalid proxy entries: ", err)
	}

	router.LoadHTMLGlob(PathTemplates + "/*")
	router.Static("/assets/", "frontend/static")

	router.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "index.gohtml", nil)
	})

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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
