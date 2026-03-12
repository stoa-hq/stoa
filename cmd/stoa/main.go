package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/stoa-hq/stoa/internal/app"
	"github.com/stoa-hq/stoa/internal/auth"
	"github.com/stoa-hq/stoa/internal/config"
	"github.com/stoa-hq/stoa/internal/database"
	"github.com/stoa-hq/stoa/internal/plugin"
	"github.com/stoa-hq/stoa/internal/seeder"
)

var (
	version    = "dev"
	configPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "stoa",
		Short: "Stoa – Headless E-Commerce System",
	}
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file path")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(adminCmd())
	rootCmd.AddCommand(seedCmd())
	rootCmd.AddCommand(pluginCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			application, err := app.New(cfg)
			if err != nil {
				return fmt.Errorf("initializing application: %w", err)
			}

			return application.Run()
		},
	}
}

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			m, err := database.NewMigrator(cfg.Database.URL, "migrations", newLogger())
			if err != nil {
				return err
			}
			defer m.Close()
			return m.Up()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			m, err := database.NewMigrator(cfg.Database.URL, "migrations", newLogger())
			if err != nil {
				return err
			}
			defer m.Close()
			return m.Down()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print current migration version",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			m, err := database.NewMigrator(cfg.Database.URL, "migrations", newLogger())
			if err != nil {
				return err
			}
			defer m.Close()
			v, dirty, err := m.Version()
			if err != nil {
				return err
			}
			fmt.Printf("Version: %d, Dirty: %v\n", v, dirty)
			return nil
		},
	})

	return cmd
}

func adminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Admin user management",
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an admin user",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			password, _ := cmd.Flags().GetString("password")

			if email == "" || password == "" {
				return fmt.Errorf("--email and --password are required")
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			logger := newLogger()

			db, err := database.New(cfg.Database, logger)
			if err != nil {
				return err
			}
			defer db.Close()

			hash, err := auth.HashPassword(password)
			if err != nil {
				return fmt.Errorf("hashing password: %w", err)
			}

			id := uuid.New()
			_, err = db.Pool.Exec(context.Background(),
				`INSERT INTO admin_users (id, email, password_hash, role, active, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, true, $5, $5)`,
				id, email, hash, string(auth.RoleSuperAdmin), time.Now())
			if err != nil {
				return fmt.Errorf("creating admin user: %w", err)
			}

			fmt.Printf("Admin user created: %s (ID: %s)\n", email, id)
			return nil
		},
	}
	createCmd.Flags().String("email", "", "Admin email address")
	createCmd.Flags().String("password", "", "Admin password")

	cmd.AddCommand(createCmd)
	return cmd
}

func seedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed the database with data",
		RunE: func(cmd *cobra.Command, args []string) error {
			demo, _ := cmd.Flags().GetBool("demo")
			if !demo {
				return fmt.Errorf("specify --demo to load demo data")
			}
			force, _ := cmd.Flags().GetBool("force")

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			logger := newLogger()

			db, err := database.New(cfg.Database, logger)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}
			defer db.Close()

			s := seeder.New(db.Pool, logger)
			if err := s.SeedDemo(context.Background(), force); err != nil {
				if errors.Is(err, seeder.ErrAlreadySeeded) {
					fmt.Fprintln(os.Stderr, "warning:", err)
					return nil
				}
				return fmt.Errorf("seeding: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().Bool("demo", false, "Load demo data")
	cmd.Flags().Bool("force", false, "Skip the already-seeded check and insert anyway")
	return cmd
}

func pluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Plugin management",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := plugin.FindModuleRoot(mustCwd())
			if err != nil {
				fmt.Println("No plugins installed.")
				return nil
			}
			imports, err := plugin.NewInstaller(root, "").ListInstalled()
			if err != nil || len(imports) == 0 {
				fmt.Println("No plugins installed.")
				return nil
			}
			for _, imp := range imports {
				fmt.Println(" •", imp)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "install <package>",
		Short: "Install a plugin (short name or full Go import path)",
		Long: `Install a plugin by short name or full Go import path.

Examples:
  stoa plugin install n8n
  stoa plugin install github.com/stoa-hq/stoa-plugins/n8n`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg := plugin.ResolvePackage(args[0])
			root, err := plugin.FindModuleRoot(mustCwd())
			if err != nil {
				return err
			}
			bin, err := resolveExecutable()
			if err != nil {
				return err
			}
			return plugin.NewInstaller(root, bin).Install(pkg)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "remove <package>",
		Short: "Remove an installed plugin (short name or full Go import path)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg := plugin.ResolvePackage(args[0])
			root, err := plugin.FindModuleRoot(mustCwd())
			if err != nil {
				return err
			}
			bin, err := resolveExecutable()
			if err != nil {
				return err
			}
			return plugin.NewInstaller(root, bin).Remove(pkg)
		},
	})

	return cmd
}

func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func resolveExecutable() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding executable path: %w", err)
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		return "", fmt.Errorf("resolving executable path: %w", err)
	}
	return bin, nil
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("stoa %s\n", version)
		},
	}
}

func newLogger() zerolog.Logger {
	return zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()
}
