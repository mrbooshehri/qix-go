package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/mrbooshehri/qix-go/internal/config"
	"github.com/mrbooshehri/qix-go/internal/storage"
	"github.com/mrbooshehri/qix-go/internal/ui"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore",
	Long:  "Create, list, and restore backups of QIX data",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a backup",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()
		store := storage.Get()
		
		// Flush any pending changes
		if err := store.FlushAll(); err != nil {
			ui.PrintWarning("Some changes may not be saved: %v", err)
		}
		
		ui.PrintInfo("Creating backup...")
		
		// Create backup filename
		timestamp := time.Now().Format("20060102_150405")
		backupName := fmt.Sprintf("qix_backup_%s.tar.gz", timestamp)
		backupPath := filepath.Join(cfg.BackupDir, backupName)
		
		// Create tar.gz archive
		if err := createTarGz(cfg.QixDir, backupPath); err != nil {
			ui.PrintError("Failed to create backup: %v", err)
			return
		}
		
		// Get backup size
		info, err := os.Stat(backupPath)
		if err != nil {
			ui.PrintError("Failed to get backup info: %v", err)
			return
		}
		
		size := float64(info.Size()) / 1024 / 1024 // MB
		
		ui.PrintSuccess("Backup created")
		ui.Cyan.Printf("  File: %s\n", backupName)
		ui.Blue.Printf("  Location: %s\n", cfg.BackupDir)
		ui.Yellow.Printf("  Size: %.2f MB\n", size)
		ui.Dim.Printf("  Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		
		// Cleanup old backups
		if _, err := cleanupOldBackups(cfg); err != nil {
			ui.PrintWarning("Failed to cleanup old backups: %v", err)
		}
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()
		
		// Find all backup files
		pattern := filepath.Join(cfg.BackupDir, "qix_backup_*.tar.gz")
		files, err := filepath.Glob(pattern)
		if err != nil {
			ui.PrintError("Failed to list backups: %v", err)
			return
		}
		
		if len(files) == 0 {
			ui.PrintEmptyState("No backups found", "Create one with: qix backup create")
			return
		}
		
		ui.PrintHeader("📦 Available Backups")
		
		// Build table
		table := ui.NewTableBuilder("Backup", "Date", "Size", "Age").
			Align(2, ui.AlignRight).
			Align(3, ui.AlignRight)
		
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			
			name := filepath.Base(file)
			modTime := info.ModTime()
			size := float64(info.Size()) / 1024 / 1024 // MB
			age := time.Since(modTime)
			
			ageStr := formatAge(age)
			
			table.Row(
				name,
				modTime.Format("2006-01-02 15:04"),
				fmt.Sprintf("%.2f MB", size),
				ageStr,
			)
		}
		
		table.PrintSimple()
		
		fmt.Println()
		ui.Dim.Printf("Backup location: %s\n", cfg.BackupDir)
		ui.Dim.Printf("Retention period: %d days\n", cfg.BackupRetentionDays)
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <backup_file>",
	Short: "Restore from a backup",
	Long:  "Restore QIX data from a backup file (creates safety backup first)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		backupFile := args[0]
		cfg := config.Get()
		
		// Find backup file
		var backupPath string
		if filepath.IsAbs(backupFile) {
			backupPath = backupFile
		} else {
			backupPath = filepath.Join(cfg.BackupDir, backupFile)
		}
		
		// Verify backup exists
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			ui.PrintError("Backup file not found: %s", backupFile)
			return
		}
		
		// Confirmation
		force, _ := cmd.Flags().GetBool("force")
		
		if !force {
			fmt.Println("⚠️  This will restore data from the backup and overwrite current data.")
			fmt.Printf("Backup: %s\n", filepath.Base(backupPath))
			fmt.Println()
			fmt.Print("Type 'restore' to confirm: ")
			
			var confirm string
			fmt.Scanln(&confirm)
			
			if confirm != "restore" {
				ui.PrintInfo("Restore cancelled")
				return
			}
		}
		
		ui.PrintInfo("Creating safety backup of current data...")
		
		// Create safety backup first
		safetyName := fmt.Sprintf("qix_backup_pre_restore_%s.tar.gz", 
			time.Now().Format("20060102_150405"))
		safetyPath := filepath.Join(cfg.BackupDir, safetyName)
		
		if err := createTarGz(cfg.QixDir, safetyPath); err != nil {
			ui.PrintError("Failed to create safety backup: %v", err)
			return
		}
		
		ui.PrintSuccess("Safety backup created: %s", safetyName)
		fmt.Println()
		
		ui.PrintInfo("Restoring from backup...")
		
		// Extract backup
		if err := extractTarGz(backupPath, filepath.Dir(cfg.QixDir)); err != nil {
			ui.PrintError("Failed to restore backup: %v", err)
			ui.PrintWarning("Your data was not modified. Safety backup: %s", safetyName)
			return
		}
		
		// Clear storage cache
		store := storage.Get()
		store.ClearCache()
		
		// Rebuild index
		if err := store.RebuildIndex(); err != nil {
			ui.PrintWarning("Failed to rebuild index: %v", err)
		}
		
		ui.PrintSuccess("Backup restored successfully")
		ui.Green.Printf("  Restored from: %s\n", filepath.Base(backupPath))
		ui.Blue.Printf("  Safety backup: %s\n", safetyName)
		fmt.Println()
		ui.Dim.Println("💡 Tip: Run 'qix doctor' to verify data integrity")
	},
}

var backupCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove old backups",
	Long:  "Delete backups older than the retention period",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()
		
		ui.PrintInfo("Cleaning up old backups (retention: %d days)...", cfg.BackupRetentionDays)
		
		count, err := cleanupOldBackups(cfg)
		if err != nil {
			ui.PrintError("Failed to cleanup backups: %v", err)
			return
		}
		
		if count == 0 {
			ui.PrintInfo("No old backups to remove")
		} else {
			ui.PrintSuccess("Removed %d old backup(s)", count)
		}
	},
}

var backupExportCmd = &cobra.Command{
	Use:   "export <output_path>",
	Short: "Export backup to a specific location",
	Long:  "Create a backup and save it to a custom location",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputPath := args[0]
		cfg := config.Get()
		store := storage.Get()
		
		// Flush changes
		if err := store.FlushAll(); err != nil {
			ui.PrintWarning("Some changes may not be saved: %v", err)
		}
		
		ui.PrintInfo("Exporting backup...")
		
		// Ensure output has .tar.gz extension
		if !strings.HasSuffix(outputPath, ".tar.gz") {
			outputPath += ".tar.gz"
		}
		
		// Create backup
		if err := createTarGz(cfg.QixDir, outputPath); err != nil {
			ui.PrintError("Failed to export backup: %v", err)
			return
		}
		
		// Get file info
		info, err := os.Stat(outputPath)
		if err != nil {
			ui.PrintError("Failed to get backup info: %v", err)
			return
		}
		
		size := float64(info.Size()) / 1024 / 1024 // MB
		
		ui.PrintSuccess("Backup exported")
		ui.Cyan.Printf("  Location: %s\n", outputPath)
		ui.Yellow.Printf("  Size: %.2f MB\n", size)
	},
}

// Helper functions

func createTarGz(sourceDir, targetFile string) error {
	// Create output file
	outFile, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer outFile.Close()
	
	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()
	
	// Walk the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip the backups directory itself
		if strings.Contains(path, "/backups/") {
			return nil
		}
		
		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		
		// Update header name to be relative
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return err
		}
		header.Name = relPath
		
		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// If not a directory, write file content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}
		
		return nil
	})
}

func extractTarGz(sourceFile, targetDir string) error {
	// Open source file
	file, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzReader)
	
	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		// Construct target path
		target := filepath.Join(targetDir, header.Name)
		
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			
		case tar.TypeReg:
			// Create file
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	
	return nil
}

func cleanupOldBackups(cfg *config.Config) (int, error) {
	pattern := filepath.Join(cfg.BackupDir, "qix_backup_*.tar.gz")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, err
	}
	
	cutoff := time.Now().AddDate(0, 0, -cfg.BackupRetentionDays)
	removed := 0
	
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				continue
			}
			removed++
		}
	}
	
	return removed, nil
}

func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	
	if days == 0 {
		return "today"
	} else if days == 1 {
		return "1 day"
	} else if days < 7 {
		return fmt.Sprintf("%d days", days)
	} else if days < 30 {
		weeks := days / 7
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	} else {
		months := days / 30
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	}
}

func init() {
	// backup restore flags
	backupRestoreCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	
	// Add subcommands
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupCleanupCmd)
	backupCmd.AddCommand(backupExportCmd)
}