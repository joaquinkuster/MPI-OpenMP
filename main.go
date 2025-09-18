package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

/*
 Ejecutar:
   go mod init formula-sim
   go get github.com/gorilla/websocket
   go run main.go

 Abre http://localhost:8080
*/

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Mensaje que viaja por websocket
type WSMessage struct {
	Type  string `json:"type"`            // "log", "done", "summary"
	Topic string `json:"topic,omitempty"` // "mpi" o "openmp"
	Text  string `json:"text,omitempty"`
	Obj   any    `json:"obj,omitempty"` // datos arbitrarios
}

// -------------------- MPI (anillo de sectores) --------------------
// Simula que un auto pasa por todos los sectores en anillo.
// En cada paso se envía por websocket: "Tiempo de sector X: Y seg"
func runMPI(sectors int, laps int, send func(WSMessage)) {
	if sectors < 1 {
		send(WSMessage{Type: "log", Topic: "mpi", Text: "Error: sectores debe ser >= 1"})
		send(WSMessage{Type: "done", Topic: "mpi"})
		return
	}
	if laps < 1 {
		laps = 1
	}

	send(WSMessage{Type: "log", Topic: "mpi", Text: fmt.Sprintf("Iniciando MPI: %d sectores, %d vueltas", sectors, laps)})

	// Simulamos 1 auto que recorre todos los sectores por vuelta.
	// Sector times típicos: 12s - 35s (ajustable)
	for lap := 1; lap <= laps; lap++ {
		send(WSMessage{Type: "log", Topic: "mpi", Text: fmt.Sprintf("=== Vuelta %d ===", lap)})
		for s := 1; s <= sectors; s++ {
			// generar tiempo en segundos con decimales
			secTime := float64(rand.Intn(2300)+1200) / 100.0 // 12.00s .. 35.99s
			// simulamos retraso real para "en tiempo real" (opcional)
			time.Sleep(300 * time.Millisecond) // pequeño delay para ver flujo en la UI
			send(WSMessage{
				Type:  "log",
				Topic: "mpi",
				Text:  fmt.Sprintf("Tiempo de sector %d: %.2f s (vuelta %d)", s, secTime, lap),
			})
		}
	}
	// Summary simple: mejor sector total
	send(WSMessage{Type: "summary", Topic: "mpi", Obj: map[string]any{"msg": "MPI finalizado"}})
	send(WSMessage{Type: "done", Topic: "mpi"})
}

// -------------------- OpenMP (vueltas rápidas entre varios autos) --------------------
type OpenMPResult struct {
	CarID    int     `json:"car_id"`
	BestLap  float64 `json:"best_lap"`
	LapCount int     `json:"lap_count"`
}

// Simula varios autos (nCars) haciendo nLaps vueltas.
// Los tiempos de vuelta estarán entre 75s y 95s típicamente (ajustable).
func runOpenMP(nCars int, nLaps int, send func(WSMessage)) {
	if nCars < 1 {
		send(WSMessage{Type: "log", Topic: "openmp", Text: "Error: cantidad de autos debe ser >= 1"})
		send(WSMessage{Type: "done", Topic: "openmp"})
		return
	}
	if nLaps < 1 {
		nLaps = 1
	}
	send(WSMessage{Type: "log", Topic: "openmp", Text: fmt.Sprintf("Iniciando OpenMP: %d autos, %d vueltas cada uno", nCars, nLaps)})

	var wg sync.WaitGroup
	results := make([]OpenMPResult, nCars)
	mutex := sync.Mutex{}

	// cada auto corre como goroutine "paralela"
	for car := 0; car < nCars; car++ {
		wg.Add(1)
		go func(carID int) {
			defer wg.Done()
			best := 1e9
			for lap := 1; lap <= nLaps; lap++ {
				// tiempo de vuelta random 75.00 - 95.99 s
				lapTime := float64(rand.Intn(2099)+7500) / 100.0
				// enviamos evento en tiempo real
				time.Sleep(200 * time.Millisecond) // pequeño delay para ver el streaming
				send(WSMessage{
					Type:  "log",
					Topic: "openmp",
					Text:  fmt.Sprintf("Auto %d - Vuelta %d: %.2f s", carID+1, lap, lapTime),
				})
				if lapTime < best {
					best = lapTime
					send(WSMessage{
						Type:  "log",
						Topic: "openmp",
						Text:  fmt.Sprintf("Auto %d - Nueva mejor vuelta: %.2f s", carID+1, best),
					})
				}
			}
			mutex.Lock()
			results[carID] = OpenMPResult{CarID: carID + 1, BestLap: best, LapCount: nLaps}
			mutex.Unlock()
		}(car)
	}

	wg.Wait()
	// calcular mejor general
	bestOverall := OpenMPResult{CarID: -1, BestLap: 1e9}
	for _, r := range results {
		if r.BestLap < bestOverall.BestLap {
			bestOverall = r
		}
	}
	send(WSMessage{Type: "summary", Topic: "openmp", Obj: map[string]any{
		"best_per_car": results,
		"best_overall": bestOverall,
	}})
	send(WSMessage{Type: "done", Topic: "openmp"})
}

// -------------------- Websocket handler --------------------

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()
	// canal para señales de cierre locales
	done := make(chan struct{})
	defer close(done)

	// función para enviar mensajes JSON seguros
	send := func(msg WSMessage) {
		// log local
		j, _ := json.Marshal(msg)
		log.Println("->", string(j))
		conn.WriteJSON(msg)
	}

	// Escuchar mensajes entrantes (start commands)
	go func() {
		for {
			var m map[string]any
			if err := conn.ReadJSON(&m); err != nil {
				log.Println("read json err:", err)
				// cerrar y terminar
				_ = conn.Close()
				return
			}
			// manejar comandos simples
			cmd, _ := m["action"].(string)
			switch cmd {
			case "start_mpi":
				sectorsF := m["sectors"]
				lapsF := m["laps"]
				sectors := int(1)
				laps := int(1)
				if v, ok := sectorsF.(float64); ok {
					sectors = int(v)
				}
				if v, ok := lapsF.(float64); ok {
					laps = int(v)
				}
				// runMPI en goroutine para no bloquear lectura
				go runMPI(sectors, laps, send)
			case "start_openmp":
				carsF := m["cars"]
				lapsF := m["laps"]
				cars := 3
				laps := 5
				if v, ok := carsF.(float64); ok {
					cars = int(v)
				}
				if v, ok := lapsF.(float64); ok {
					laps = int(v)
				}
				go runOpenMP(cars, laps, send)
			default:
				send(WSMessage{Type: "log", Text: fmt.Sprintf("Comando no reconocido: %v", cmd)})
			}
		}
	}()

	// Mantener conexión abierta hasta que cliente la cierre
	<-done
}

// -------------------- HTTP (sirve HTML/JS) --------------------

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	indexTmpl.Execute(w, nil)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ws", wsHandler)

	addr := ":8080"
	fmt.Println("Servidor en http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// -------------------- HTML + JS embebido --------------------

const indexHTML = `
<!doctype html>
<html>
<head>
  <meta charset="utf-8"/>
  <title>Simulaciones MPI / OpenMP - Formula1</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 16px; }
    .col { display:inline-block; vertical-align:top; margin-right:20px; width:45%; }
    textarea{ width:100%; height:300px; }
    input[type="number"]{ width:80px; }
    button{ padding:8px 12px; margin-top:6px; }
    .log-mpi{ background:#f0f8ff; padding:8px; border-radius:6px; height:320px; overflow:auto;}
    .log-openmp{ background:#fff8f0; padding:8px; border-radius:6px; height:320px; overflow:auto;}
  </style>
</head>
<body>
  <h2>Simulaciones Formula1 — MPI (sectores) y OpenMP (vueltas rápidas)</h2>
  <div style="display:flex; gap: 16px;">
    <div class="col">
      <h3>MPI - Sectores (anillo)</h3>
      <label>Sector count: <input id="mpi-sectors" type="number" value="5" min="1"></label><br>
      <label>Vueltas: <input id="mpi-laps" type="number" value="3" min="1"></label><br>
      <button id="start-mpi">Iniciar MPI</button>
      <div style="margin-top:10px;">
        <h4>Salida MPI</h4>
        <div id="mpi-log" class="log-mpi"></div>
      </div>
    </div>

    <div class="col">
      <h3>OpenMP - Vueltas rápidas</h3>
      <label>Autos: <input id="openmp-cars" type="number" value="4" min="1"></label><br>
      <label>Vueltas por auto: <input id="openmp-laps" type="number" value="5" min="1"></label><br>
      <button id="start-openmp">Iniciar OpenMP</button>
      <div style="margin-top:10px;">
        <h4>Salida OpenMP</h4>
        <div id="openmp-log" class="log-openmp"></div>
      </div>
    </div>
  </div>

  <script>
    const ws = new WebSocket("ws://" + location.host + "/ws");
    const mpiLog = document.getElementById("mpi-log");
    const openmpLog = document.getElementById("openmp-log");

    ws.onopen = () => {
      appendBoth("Conexión WebSocket establecida.");
    };
    ws.onclose = () => appendBoth("WebSocket cerrado.");
    ws.onerror = (e) => appendBoth("WebSocket error: " + e);

    ws.onmessage = (evt) => {
      try {
        const msg = JSON.parse(evt.data);
        // msg: {type, topic, text, obj}
        if (msg.topic === "mpi") {
          append(mpiLog, formatMessage(msg));
        } else if (msg.topic === "openmp") {
          append(openmpLog, formatMessage(msg));
        } else {
          // general
          appendBoth(formatMessage(msg));
        }

        if (msg.type === "summary") {
          if (msg.topic === "openmp") {
            append(openmpLog, "<b>Resumen OpenMP:</b> " + JSON.stringify(msg.obj));
          } else if (msg.topic === "mpi") {
            append(mpiLog, "<b>Resumen MPI:</b> " + JSON.stringify(msg.obj));
          }
        }
      } catch (err) {
        appendBoth("Mensaje no JSON: " + evt.data);
      }
    };

    function formatMessage(msg) {
      if (msg.type === "log") return sanitize(msg.text);
      if (msg.type === "done") return "<i>Proceso finalizado (" + (msg.topic||"") + ")</i>";
      if (msg.type === "summary") return "<i>Resumen: " + JSON.stringify(msg.obj) + "</i>";
      return JSON.stringify(msg);
    }

    function append(target, text) {
      const p = document.createElement("div");
      p.innerHTML = text;
      target.appendChild(p);
      target.scrollTop = target.scrollHeight;
    }
    function appendBoth(text) {
      append(mpiLog, text);
      append(openmpLog, text);
    }
    function sanitize(s) {
      if (!s) return "";
      return s.replace(/</g, "&lt;").replace(/>/g, "&gt;");
    }

    document.getElementById("start-mpi").onclick = () => {
      const sectors = parseInt(document.getElementById("mpi-sectors").value) || 5;
      const laps = parseInt(document.getElementById("mpi-laps").value) || 3;
      ws.send(JSON.stringify({ action: "start_mpi", sectors: sectors, laps: laps }));
      append(mpiLog, "<b>Comando enviado: iniciar MPI</b>");
    };

    document.getElementById("start-openmp").onclick = () => {
      const cars = parseInt(document.getElementById("openmp-cars").value) || 4;
      const laps = parseInt(document.getElementById("openmp-laps").value) || 5;
      ws.send(JSON.stringify({ action: "start_openmp", cars: cars, laps: laps }));
      append(openmpLog, "<b>Comando enviado: iniciar OpenMP</b>");
    };
  </script>
</body>
</html>
`
