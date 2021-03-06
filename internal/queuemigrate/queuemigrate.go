package queuemigrate

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/brocaar/lora-app-server/internal/downlink"

	"github.com/brocaar/lora-app-server/internal/common"
	"github.com/brocaar/lora-app-server/internal/storage"
	"github.com/brocaar/lorawan"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// DeviceQueueItem represents an item in the device queue (downlink).
type DeviceQueueItem struct {
	ID        int64         `db:"id"`
	CreatedAt time.Time     `db:"created_at"`
	UpdatedAt time.Time     `db:"updated_at"`
	Reference string        `db:"reference"`
	DevEUI    lorawan.EUI64 `db:"dev_eui"`
	Confirmed bool          `db:"confirmed"`
	Pending   bool          `db:"pending"`
	FPort     uint8         `db:"fport"`
	Data      []byte        `db:"data"`
}

// StartDeviceQueueMigration starts the migration of all device-queues.
func StartDeviceQueueMigration() error {
	var items []DeviceQueueItem
	err := sqlx.Select(common.DB, &items, "select * from device_queue where pending = false order by id")
	if err != nil {
		return errors.Wrap(err, "select error")
	}

	for _, qi := range items {
		err = migrateQueueItem(qi)
		if err != nil {
			return errors.Wrap(err, "migrate queue-item error")
		}
	}

	return nil
}

func migrateQueueItem(qi DeviceQueueItem) error {
	return storage.Transaction(common.DB, func(tx *sqlx.Tx) error {
		if err := deleteQueueItem(tx, qi.ID); err != nil {
			return errors.Wrap(err, "delete device-queue item error")
		}

		if err := downlink.EnqueueDownlinkPayload(tx, qi.DevEUI, qi.Reference, qi.Confirmed, qi.FPort, qi.Data); err != nil {
			if grpc.Code(errors.Cause(err)) == codes.NotFound {
				return nil
			}
			return errors.Wrap(err, "enqueue downlink payload error")
		}

		return nil
	})
}

func deleteQueueItem(db sqlx.Execer, id int64) error {
	_, err := db.Exec("delete from device_queue where id = $1", id)
	if err != nil {
		return errors.Wrap(err, "delete error")
	}
	return nil
}
