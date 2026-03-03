package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/cors"

	"github.com/swatkatz/advisorhub/backend/internal/account"
	"github.com/swatkatz/advisorhub/backend/internal/actionitem"
	"github.com/swatkatz/advisorhub/backend/internal/alert"
	"github.com/swatkatz/advisorhub/backend/internal/client"
	"github.com/swatkatz/advisorhub/backend/internal/contribution"
	"github.com/swatkatz/advisorhub/backend/internal/eventbus"
	"github.com/swatkatz/advisorhub/backend/internal/graph"
	"github.com/swatkatz/advisorhub/backend/internal/seed"
	"github.com/swatkatz/advisorhub/backend/internal/temporal"
	"github.com/swatkatz/advisorhub/backend/internal/transfer"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Timezone: EST for all timestamps (prototype simplification).
	est, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatalf("loading timezone: %v", err)
	}
	now := func() time.Time { return time.Now().In(est) }

	// Database connection.
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer db.Close()

	// Event bus.
	bus := eventbus.New()

	// Repositories.
	clientRepo := client.NewPostgresClientRepo(db)
	householdRepo := client.NewPostgresHouseholdRepo(db)
	goalRepo := client.NewPostgresGoalRepo(db)
	noteRepo := client.NewPostgresNoteRepo(db)
	advisorRepo := client.NewPostgresAdvisorRepo(db)
	accountRepo := account.NewPostgresAccountRepo(db)
	respBenRepo := account.NewPostgresRESPBeneficiaryRepo(db)
	transferRepo := transfer.NewPostgresTransferRepo(db, now)
	contribRepo := contribution.NewPostgresContributionRepo(db)

	// Services.
	actionItemRepo := actionitem.NewPostgresActionItemRepo(db)
	actionItemSvc := actionitem.NewService(actionItemRepo)

	contribEngine := contribution.NewEngine(contribRepo, accountRepo, respBenRepo, bus)
	transferMon := transfer.NewMonitor(transferRepo, bus)
	temporalScanner := temporal.NewScanner(clientRepo, accountRepo, respBenRepo, contribEngine, bus)

	// Alert service with enhancer.
	alertRepo := alert.NewPostgresAlertRepository(db)
	var enhancer alert.Enhancer
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey != "" {
		enhancer = &alert.ClaudeEnhancer{
			Clients: clientRepo,
			Notes:   noteRepo,
			APIKey:  apiKey,
		}
	} else {
		enhancer = &alert.StubEnhancer{}
		log.Println("ANTHROPIC_API_KEY not set, using stub enhancer")
	}
	alertSvc := alert.NewAlertService(alertRepo, bus, actionItemSvc, enhancer, now)

	// Start alert event listener — subscribes to all domain events and calls ProcessEvent.
	go startAlertListener(bus, alertSvc)

	// Seed data.
	seedLoader := seed.New(seed.Deps{DB: db, Bus: bus})
	if err := seedLoader.Load(context.Background()); err != nil {
		log.Printf("seed data: %v", err)
	}

	// GraphQL resolver.
	resolver := &graph.Resolver{
		ClientRepo:        clientRepo,
		HouseholdRepo:     householdRepo,
		GoalRepo:          goalRepo,
		NoteRepo:          noteRepo,
		AdvisorRepo:       advisorRepo,
		AccountRepo:       accountRepo,
		AlertService:      alertSvc,
		AlertRepo:         alertRepo,
		ActionItemService: actionItemSvc,
		ContribEngine:     contribEngine,
		TransferRepo:      transferRepo,
		TransferMonitor:   transferMon,
		TemporalScanner:   temporalScanner,
		EventBus:          bus,
	}

	// GraphQL handler with SSE transport.
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Transport order matters: SSE first for Railway proxy compatibility.
	srv.AddTransport(transport.SSE{
		KeepAlivePingInterval: 15 * time.Second,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	// CORS.
	allowedOrigins := []string{"http://localhost:5173"} // Vite dev server
	if corsOrigin := os.Getenv("CORS_ALLOWED_ORIGIN"); corsOrigin != "" {
		for _, origin := range strings.Split(corsOrigin, ",") {
			if o := strings.TrimSpace(origin); o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
	})

	// SSE-specific response headers for Railway proxy compatibility.
	sseHeaders := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "text/event-stream" {
			w.Header().Set("X-Accel-Buffering", "no")
			w.Header().Set("Cache-Control", "no-cache, no-transform")
			w.Header().Set("Connection", "keep-alive")
		}
		srv.ServeHTTP(w, r)
	})

	// Routes.
	mux := http.NewServeMux()
	mux.Handle("/graphql", c.Handler(sseHeaders))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("server listening on :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux))
}

// startAlertListener subscribes to all domain event types and calls alertService.ProcessEvent.
func startAlertListener(bus eventbus.EventBus, svc alert.AlertService) {
	eventTypes := []string{
		alert.EventOverContributionDetected,
		alert.EventCESGGap,
		alert.EventTransferStuck,
		alert.EventDeadlineApproaching,
		alert.EventAgeMilestone,
		alert.EventEngagementStale,
		alert.EventCashUninvested,
		alert.EventPortfolioDrift,
		alert.EventTaxLossOpportunity,
		alert.EventTransferCompleted,
		alert.EventContributionProcessed,
		alert.EventDividendReceived,
	}

	channels := make([]<-chan eventbus.EventEnvelope, len(eventTypes))
	for i, et := range eventTypes {
		channels[i] = bus.Subscribe(et)
	}

	ctx := context.Background()
	for {
		for _, ch := range channels {
			select {
			case env := <-ch:
				if err := svc.ProcessEvent(ctx, env); err != nil {
					log.Printf("alert processing error for %s: %v", env.Type, err)
				}
			default:
			}
		}
		// Small sleep to avoid busy-spinning.
		time.Sleep(10 * time.Millisecond)
	}
}
