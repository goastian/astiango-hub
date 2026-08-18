package cmd

import (
	"fmt"
	"strings"

	"github.com/goastian/astiango-hub/core/models/models"
	modelservice "github.com/goastian/astiango-hub/core/models/service"
	coremongo "github.com/goastian/astiango-hub/core/mongo"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

var applyEncryptionMigration bool

func init() {
	migrateEncryptionCmd.Flags().BoolVar(&applyEncryptionMigration, "apply", false, "write migrated values (the default only reports candidates)")
	rootCmd.AddCommand(migrateEncryptionCmd)
}

var migrateEncryptionCmd = &cobra.Command{
	Use:   "migrate-encryption",
	Short: "Migrate stored database credentials to AES-256-GCM",
	RunE: func(cmd *cobra.Command, args []string) error {
		collection := coremongo.GetMongoCol(modelservice.GetCollectionNameByInstance(models.Database{}))
		var databases []models.Database
		err := collection.Find(bson.M{"encrypted_password": bson.M{"$exists": true, "$ne": ""}}, nil).All(&databases)
		if err != nil {
			return fmt.Errorf("find encrypted database credentials: %w", err)
		}

		candidates, migrated := 0, 0
		for _, database := range databases {
			if strings.HasPrefix(database.EncryptedPassword, "agcm:v1:") {
				continue
			}
			candidates++
			if !applyEncryptionMigration {
				continue
			}
			ciphertext, changed, err := utils.ReencryptAES(database.EncryptedPassword)
			if err != nil {
				return fmt.Errorf("migrate database credential %s: %w", database.Id.Hex(), err)
			}
			if !changed {
				continue
			}
			if err := collection.UpdateId(database.Id, bson.M{"$set": bson.M{"encrypted_password": ciphertext}}); err != nil {
				return fmt.Errorf("store migrated database credential %s: %w", database.Id.Hex(), err)
			}
			migrated++
		}
		if applyEncryptionMigration {
			cmd.Printf("Migrated %d of %d legacy encrypted database credential(s).\n", migrated, candidates)
		} else {
			cmd.Printf("Found %d legacy encrypted database credential(s). Re-run with --apply after backing up the database.\n", candidates)
		}
		return nil
	},
}
