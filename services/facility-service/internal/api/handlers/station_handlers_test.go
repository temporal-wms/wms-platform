package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/wms-platform/shared/pkg/errors"
	"github.com/wms-platform/shared/pkg/logging"

	"github.com/wms-platform/facility-service/internal/application"
)

type mockStationService struct {
	createStationFn    func(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error)
	getStationFn       func(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error)
	updateStationFn    func(ctx context.Context, cmd application.UpdateStationCommand) (*application.StationDTO, error)
	deleteStationFn    func(ctx context.Context, cmd application.DeleteStationCommand) error
	setCapabilitiesFn  func(ctx context.Context, cmd application.SetCapabilitiesCommand) (*application.StationDTO, error)
	addCapabilityFn    func(ctx context.Context, cmd application.AddCapabilityCommand) (*application.StationDTO, error)
	removeCapabilityFn func(ctx context.Context, cmd application.RemoveCapabilityCommand) (*application.StationDTO, error)
	setStatusFn        func(ctx context.Context, cmd application.SetStationStatusCommand) (*application.StationDTO, error)
	findCapableFn      func(ctx context.Context, query application.FindCapableStationsQuery) ([]application.StationDTO, error)
	listStationsFn     func(ctx context.Context, query application.ListStationsQuery) ([]application.StationDTO, error)
	getByZoneFn        func(ctx context.Context, query application.GetStationsByZoneQuery) ([]application.StationDTO, error)
	getByTypeFn        func(ctx context.Context, query application.GetStationsByTypeQuery) ([]application.StationDTO, error)
	getByStatusFn      func(ctx context.Context, query application.GetStationsByStatusQuery) ([]application.StationDTO, error)
}

func (m *mockStationService) CreateStation(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error) {
	if m.createStationFn == nil {
		panic("CreateStation not implemented")
	}
	return m.createStationFn(ctx, cmd)
}

func (m *mockStationService) GetStation(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error) {
	if m.getStationFn == nil {
		panic("GetStation not implemented")
	}
	return m.getStationFn(ctx, query)
}

func (m *mockStationService) UpdateStation(ctx context.Context, cmd application.UpdateStationCommand) (*application.StationDTO, error) {
	if m.updateStationFn == nil {
		panic("UpdateStation not implemented")
	}
	return m.updateStationFn(ctx, cmd)
}

func (m *mockStationService) DeleteStation(ctx context.Context, cmd application.DeleteStationCommand) error {
	if m.deleteStationFn == nil {
		panic("DeleteStation not implemented")
	}
	return m.deleteStationFn(ctx, cmd)
}

func (m *mockStationService) SetCapabilities(ctx context.Context, cmd application.SetCapabilitiesCommand) (*application.StationDTO, error) {
	if m.setCapabilitiesFn == nil {
		panic("SetCapabilities not implemented")
	}
	return m.setCapabilitiesFn(ctx, cmd)
}

func (m *mockStationService) AddCapability(ctx context.Context, cmd application.AddCapabilityCommand) (*application.StationDTO, error) {
	if m.addCapabilityFn == nil {
		panic("AddCapability not implemented")
	}
	return m.addCapabilityFn(ctx, cmd)
}

func (m *mockStationService) RemoveCapability(ctx context.Context, cmd application.RemoveCapabilityCommand) (*application.StationDTO, error) {
	if m.removeCapabilityFn == nil {
		panic("RemoveCapability not implemented")
	}
	return m.removeCapabilityFn(ctx, cmd)
}

func (m *mockStationService) SetStatus(ctx context.Context, cmd application.SetStationStatusCommand) (*application.StationDTO, error) {
	if m.setStatusFn == nil {
		panic("SetStatus not implemented")
	}
	return m.setStatusFn(ctx, cmd)
}

func (m *mockStationService) FindCapableStations(ctx context.Context, query application.FindCapableStationsQuery) ([]application.StationDTO, error) {
	if m.findCapableFn == nil {
		panic("FindCapableStations not implemented")
	}
	return m.findCapableFn(ctx, query)
}

func (m *mockStationService) ListStations(ctx context.Context, query application.ListStationsQuery) ([]application.StationDTO, error) {
	if m.listStationsFn == nil {
		panic("ListStations not implemented")
	}
	return m.listStationsFn(ctx, query)
}

func (m *mockStationService) GetByZone(ctx context.Context, query application.GetStationsByZoneQuery) ([]application.StationDTO, error) {
	if m.getByZoneFn == nil {
		panic("GetByZone not implemented")
	}
	return m.getByZoneFn(ctx, query)
}

func (m *mockStationService) GetByType(ctx context.Context, query application.GetStationsByTypeQuery) ([]application.StationDTO, error) {
	if m.getByTypeFn == nil {
		panic("GetByType not implemented")
	}
	return m.getByTypeFn(ctx, query)
}

func (m *mockStationService) GetByStatus(ctx context.Context, query application.GetStationsByStatusQuery) ([]application.StationDTO, error) {
	if m.getByStatusFn == nil {
		panic("GetByStatus not implemented")
	}
	return m.getByStatusFn(ctx, query)
}

func newTestRouter(service StationService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := logging.New(logging.DefaultConfig("test"))
	handlers := NewStationHandlers(service, logger, nil)
	handlers.RegisterRoutes(router.Group("/api/v1"))
	return router
}

func performRequest(router *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestStationHandlers_CreateStation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := &mockStationService{
			createStationFn: func(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error) {
				if cmd.StationID != "STN-1" {
					t.Fatalf("StationID = %s", cmd.StationID)
				}
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		body := `{"stationId":"STN-1","name":"Station","zone":"A","stationType":"packing","capabilities":["c1"],"maxConcurrentTasks":3}`
		rec := performRequest(router, http.MethodPost, "/api/v1/stations", body)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"stationId":"STN-1"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("bad json", func(t *testing.T) {
		service := &mockStationService{
			createStationFn: func(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error) {
				return nil, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPost, "/api/v1/stations", `{"stationId":}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("app error", func(t *testing.T) {
		service := &mockStationService{
			createStationFn: func(ctx context.Context, cmd application.CreateStationCommand) (*application.StationDTO, error) {
				return nil, errors.ErrValidation("bad")
			},
		}
		router := newTestRouter(service)
		body := `{"stationId":"STN-1","name":"Station","zone":"A","stationType":"packing"}`
		rec := performRequest(router, http.MethodPost, "/api/v1/stations", body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}

func TestStationHandlers_GetStation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := &mockStationService{
			getStationFn: func(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error) {
				if query.StationID != "STN-2" {
					t.Fatalf("StationID = %s", query.StationID)
				}
				return &application.StationDTO{StationID: query.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodGet, "/api/v1/stations/STN-2", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("app error", func(t *testing.T) {
		service := &mockStationService{
			getStationFn: func(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error) {
				return nil, errors.ErrNotFound("station")
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodGet, "/api/v1/stations/STN-404", "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		service := &mockStationService{
			getStationFn: func(ctx context.Context, query application.GetStationQuery) (*application.StationDTO, error) {
				return nil, fmt.Errorf("boom")
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodGet, "/api/v1/stations/STN-500", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}

func TestStationHandlers_UpdateDelete(t *testing.T) {
	t.Run("update success", func(t *testing.T) {
		service := &mockStationService{
			updateStationFn: func(ctx context.Context, cmd application.UpdateStationCommand) (*application.StationDTO, error) {
				if cmd.StationID != "STN-3" || cmd.Zone != "B" {
					t.Fatalf("unexpected command: %+v", cmd)
				}
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPut, "/api/v1/stations/STN-3", `{"zone":"B"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("update bad json", func(t *testing.T) {
		service := &mockStationService{
			updateStationFn: func(ctx context.Context, cmd application.UpdateStationCommand) (*application.StationDTO, error) {
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPut, "/api/v1/stations/STN-3", `{"zone":}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		service := &mockStationService{
			deleteStationFn: func(ctx context.Context, cmd application.DeleteStationCommand) error {
				if cmd.StationID != "STN-3" {
					t.Fatalf("StationID = %s", cmd.StationID)
				}
				return nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodDelete, "/api/v1/stations/STN-3", "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("delete app error", func(t *testing.T) {
		service := &mockStationService{
			deleteStationFn: func(ctx context.Context, cmd application.DeleteStationCommand) error {
				return errors.ErrNotFound("station")
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodDelete, "/api/v1/stations/STN-404", "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}

func TestStationHandlers_CapabilitiesAndStatus(t *testing.T) {
	t.Run("set capabilities", func(t *testing.T) {
		service := &mockStationService{
			setCapabilitiesFn: func(ctx context.Context, cmd application.SetCapabilitiesCommand) (*application.StationDTO, error) {
				if cmd.StationID != "STN-4" || len(cmd.Capabilities) != 2 {
					t.Fatalf("unexpected command: %+v", cmd)
				}
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPut, "/api/v1/stations/STN-4/capabilities", `{"capabilities":["c1","c2"]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("add capability", func(t *testing.T) {
		service := &mockStationService{
			addCapabilityFn: func(ctx context.Context, cmd application.AddCapabilityCommand) (*application.StationDTO, error) {
				if cmd.StationID != "STN-4" || cmd.Capability != "c3" {
					t.Fatalf("unexpected command: %+v", cmd)
				}
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPost, "/api/v1/stations/STN-4/capabilities/c3", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("remove capability", func(t *testing.T) {
		service := &mockStationService{
			removeCapabilityFn: func(ctx context.Context, cmd application.RemoveCapabilityCommand) (*application.StationDTO, error) {
				if cmd.StationID != "STN-4" || cmd.Capability != "c1" {
					t.Fatalf("unexpected command: %+v", cmd)
				}
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodDelete, "/api/v1/stations/STN-4/capabilities/c1", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("set status bad json", func(t *testing.T) {
		service := &mockStationService{
			setStatusFn: func(ctx context.Context, cmd application.SetStationStatusCommand) (*application.StationDTO, error) {
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPut, "/api/v1/stations/STN-4/status", `{"status":}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("set status success", func(t *testing.T) {
		service := &mockStationService{
			setStatusFn: func(ctx context.Context, cmd application.SetStationStatusCommand) (*application.StationDTO, error) {
				if cmd.Status != "active" {
					t.Fatalf("status = %s", cmd.Status)
				}
				return &application.StationDTO{StationID: cmd.StationID}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPut, "/api/v1/stations/STN-4/status", `{"status":"active"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}

func TestStationHandlers_Queries(t *testing.T) {
	t.Run("find capable stations", func(t *testing.T) {
		service := &mockStationService{
			findCapableFn: func(ctx context.Context, query application.FindCapableStationsQuery) ([]application.StationDTO, error) {
				if len(query.Requirements) != 2 || query.Zone != "Z1" {
					t.Fatalf("unexpected query: %+v", query)
				}
				return []application.StationDTO{{StationID: "STN-5"}}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPost, "/api/v1/stations/find-capable", `{"requirements":["c1","c2"],"zone":"Z1"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("find capable bad json", func(t *testing.T) {
		service := &mockStationService{
			findCapableFn: func(ctx context.Context, query application.FindCapableStationsQuery) ([]application.StationDTO, error) {
				return nil, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodPost, "/api/v1/stations/find-capable", `{"requirements":}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("list stations", func(t *testing.T) {
		service := &mockStationService{
			listStationsFn: func(ctx context.Context, query application.ListStationsQuery) ([]application.StationDTO, error) {
				if query.Limit != 10 || query.Offset != 5 {
					t.Fatalf("unexpected query: %+v", query)
				}
				return []application.StationDTO{{StationID: "STN-6"}}, nil
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodGet, "/api/v1/stations?limit=10&offset=5", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"stationId":"STN-6"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("list stations error", func(t *testing.T) {
		service := &mockStationService{
			listStationsFn: func(ctx context.Context, query application.ListStationsQuery) ([]application.StationDTO, error) {
				return nil, fmt.Errorf("list failed")
			},
		}
		router := newTestRouter(service)
		rec := performRequest(router, http.MethodGet, "/api/v1/stations", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("get by zone/type/status", func(t *testing.T) {
		service := &mockStationService{
			getByZoneFn: func(ctx context.Context, query application.GetStationsByZoneQuery) ([]application.StationDTO, error) {
				if query.Zone != "A1" {
					t.Fatalf("unexpected query: %+v", query)
				}
				return []application.StationDTO{}, nil
			},
			getByTypeFn: func(ctx context.Context, query application.GetStationsByTypeQuery) ([]application.StationDTO, error) {
				if query.StationType != "packing" {
					t.Fatalf("unexpected query: %+v", query)
				}
				return []application.StationDTO{}, nil
			},
			getByStatusFn: func(ctx context.Context, query application.GetStationsByStatusQuery) ([]application.StationDTO, error) {
				if query.Status != "active" {
					t.Fatalf("unexpected query: %+v", query)
				}
				return []application.StationDTO{}, nil
			},
		}
		router := newTestRouter(service)

		rec := performRequest(router, http.MethodGet, "/api/v1/stations/zone/A1", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("zone status = %d", rec.Code)
		}

		rec = performRequest(router, http.MethodGet, "/api/v1/stations/type/packing", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("type status = %d", rec.Code)
		}

		rec = performRequest(router, http.MethodGet, "/api/v1/stations/status/active", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status status = %d", rec.Code)
		}
	})

	t.Run("get by zone/type/status errors", func(t *testing.T) {
		service := &mockStationService{
			getByZoneFn: func(ctx context.Context, query application.GetStationsByZoneQuery) ([]application.StationDTO, error) {
				return nil, fmt.Errorf("zone failed")
			},
			getByTypeFn: func(ctx context.Context, query application.GetStationsByTypeQuery) ([]application.StationDTO, error) {
				return nil, fmt.Errorf("type failed")
			},
			getByStatusFn: func(ctx context.Context, query application.GetStationsByStatusQuery) ([]application.StationDTO, error) {
				return nil, fmt.Errorf("status failed")
			},
		}
		router := newTestRouter(service)

		rec := performRequest(router, http.MethodGet, "/api/v1/stations/zone/A1", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("zone status = %d", rec.Code)
		}

		rec = performRequest(router, http.MethodGet, "/api/v1/stations/type/packing", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("type status = %d", rec.Code)
		}

		rec = performRequest(router, http.MethodGet, "/api/v1/stations/status/active", "")
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status status = %d", rec.Code)
		}
	})
}
