package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	botToken   string
	startTime  time.Time
)

func main() {
	// Carrega variáveis de ambiente do .env (opcional)
	godotenv.Load()

	// Pega o token do bot
	botToken = os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("❌ DISCORD_BOT_TOKEN não encontrado! Configure no .env ou nas variáveis do Render")
	}

	// Marca o tempo de inicialização
	startTime = time.Now()

	// Cria a sessão do Discord
	session, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatalf("❌ Erro ao criar sessão: %v", err)
	}

	// Define os intents (necessário pra receber mensagens)
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// Adiciona o handler de mensagens
	session.AddHandler(messageHandler)

	// Abre a conexão WebSocket com o Discord
	err = session.Open()
	if err != nil {
		log.Fatalf("❌ Erro ao abrir conexão: %v", err)
	}
	defer session.Close()

	fmt.Println("✅ Bot está online!")
	fmt.Printf("🤖 Conectado como: %s\n", session.State.User.Username)
	fmt.Printf("📡 Latência do WebSocket: %v\n", session.HeartbeatLatency())

	// Health check pro Render (opcional, mas bom ter)
	go healthCheckServer()

	// Mantém o bot rodando até receber SIGINT ou SIGTERM
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("\n🛑 Bot desligando...")
}

// Handler principal de mensagens
func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignora mensagens do próprio bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Comando: !ping
	if m.Content == "!ping" {
		// Calcula a latência
		latency := s.HeartbeatLatency()

		// Envia a resposta com o ping
		response := fmt.Sprintf("🏓 Pong!\n📡 Latência: **%v**", latency)
		
		// Opcional: timestamp do servidor
		response += fmt.Sprintf("\n⏰ Servidor rodando há: **%s**", time.Since(startTime).Round(time.Second))

		_, err := s.ChannelMessageSend(m.ChannelID, response)
		if err != nil {
			log.Printf("❌ Erro ao enviar mensagem: %v", err)
		}
	}

	// Comando: !ping <mensagem> (pra testar latência de volta)
	if m.Content == "!pong" {
		_, err := s.ChannelMessageSend(m.ChannelID, "🏓 Ping!")
		if err != nil {
			log.Printf("❌ Erro ao enviar mensagem: %v", err)
		}
	}
}

// Servidor HTTP pra health check (Render exige uma porta exposta)
func healthCheckServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "online",
			"uptime":  time.Since(startTime).String(),
			"bot":     "Discord Ping Bot",
			"version": "1.0.0",
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🌐 Health check server rodando na porta %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
