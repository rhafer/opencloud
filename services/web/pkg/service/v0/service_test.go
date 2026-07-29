package svc

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/web/mocks"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

func TestManagedAnnouncement(t *testing.T) {
	newWeb := func(store *announcement.Store) Web {
		return Web{logger: log.NopLogger(), announcementStore: store}
	}
	// storeReturning builds a store whose backing bucket returns the given JSON for the announcement key.
	storeReturning := func(t *testing.T, value string) *announcement.Store {
		entry := mocks.NewKeyValueEntry(t)
		entry.EXPECT().Value().Return([]byte(value))
		kv := mocks.NewKeyValue(t)
		kv.EXPECT().Get(mock.Anything, "announcement").Return(entry, nil)
		return announcement.NewStore(kv)
	}
	// emptyStore builds a store whose backing bucket has no announcement.
	emptyStore := func(t *testing.T) *announcement.Store {
		kv := mocks.NewKeyValue(t)
		kv.EXPECT().Get(mock.Anything, "announcement").Return(nil, jetstream.ErrKeyNotFound)
		return announcement.NewStore(kv)
	}

	t.Run("not managed when there is no store, so a static config is kept", func(t *testing.T) {
		a, managed := newWeb(nil).managedAnnouncement(context.Background())
		require.Nil(t, a)
		require.False(t, managed)
	})

	t.Run("not managed when the store is empty, so a static config is kept", func(t *testing.T) {
		a, managed := newWeb(emptyStore(t)).managedAnnouncement(context.Background())
		require.Nil(t, a)
		require.False(t, managed)
	})

	t.Run("managed but nil when disabled, so a static config is cleared", func(t *testing.T) {
		s := storeReturning(t, `{"enabled":false,"bannerText":"hi","infoText":"info"}`)
		a, managed := newWeb(s).managedAnnouncement(context.Background())
		require.Nil(t, a)
		require.True(t, managed)
	})

	t.Run("not managed when enabled but the banner text is empty", func(t *testing.T) {
		s := storeReturning(t, `{"enabled":true,"bannerText":"","infoText":"info"}`)
		a, managed := newWeb(s).managedAnnouncement(context.Background())
		require.Nil(t, a)
		require.False(t, managed)
	})

	t.Run("managed with banner and info text when enabled with a banner text", func(t *testing.T) {
		s := storeReturning(t, `{"enabled":true,"bannerText":"hi","infoText":"info"}`)
		a, managed := newWeb(s).managedAnnouncement(context.Background())
		require.Equal(t, &config.Announcement{BannerText: "hi", InfoText: "info"}, a)
		require.True(t, managed)
	})
}
