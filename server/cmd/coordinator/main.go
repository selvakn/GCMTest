// Command coordinator runs the push-notification-latency coordinator: it
// enrolls devices, triggers rounds, records receipts, and serves latency /
// success-rate reports. See specs/001-push-notification-latency/.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/selvakn/gcmtest/server/internal/api"
	"github.com/selvakn/gcmtest/server/internal/config"
	"github.com/selvakn/gcmtest/server/internal/push"
	"github.com/selvakn/gcmtest/server/internal/store"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := store.Open(cfg.DBPath, cfg.MigrationsDir)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	fcmSender, err := push.NewFCMSender(ctx, cfg.FCMServiceAccountJSON)
	if err != nil {
		log.Fatalf("init FCM sender: %v", err)
	}
	senders := push.Registry{"android": fcmSender}

	deviceStore := store.NewDeviceStore(db)
	roundStore := store.NewRoundStore(db)
	deliveryStore := store.NewDeliveryStore(db)
	reportStore := store.NewReportStore(db)

	handlers := api.Handlers{
		OperatorToken: cfg.OperatorToken,
		Devices:       &api.DevicesHandler{Devices: deviceStore},
		Rounds:        &api.RoundsHandler{Rounds: roundStore, Sender: senders},
		Receipts:      &api.ReceiptsHandler{Deliveries: deliveryStore},
		Reports:       &api.ReportsHandler{Deliveries: deliveryStore, Reports: reportStore},
	}
	mux := api.NewRouter(handlers)

	log.Printf("coordinator listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
