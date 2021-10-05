package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func migrate3xTitlesToSFTPGo() {
	if !conf.SFTPGoEnabled {
		return
	}

	var titles, err = models.Titles()
	if err != nil {
		logger.Fatalf("Unable to perform pre-init title scan: %s", err)
	}

	for _, title := range titles {
		if !title.SFTPConnected && title.SFTPUser != "" && title.LegacyPass != "" {
			logger.Infof("Connecting title %s (%s) to use SFTPGo...", title.Name, title.SFTPUser)
			_migrationCreateSFTPGoTitle(title)
		}
	}
}

func _migrationCreateSFTPGoTitle(t *models.Title) error {
	// We connect to SFTPGo, we we need a transaction
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()

	t.SFTPConnected = true
	var err = t.SaveOp(op)
	if err == nil {
		_, err = dbi.SFTP.CreateUser(t.SFTPUser, t.LegacyPass, int64(conf.SFTPGoNewUserQuota), t.Name+" / "+t.LCCN)
	}
	if err != nil {
		op.Rollback()
		t.SFTPConnected = false
		return err
	}

	op.EndTransaction()
	return op.Err()
}
