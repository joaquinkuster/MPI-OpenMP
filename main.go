package main

import (
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
Simulaciones Fórmula 1 con MPI y OpenMP (Didáctico)
---------------------------------------------------
MPI -> Simula un anillo de sectores con paso de mensajes entre goroutines usando canales
OpenMP -> Simula varios autos corriendo vueltas rápidas en paralelo usando goroutines y mutex
*/

// -------------------- Configuración WebSocket --------------------

// Permite cualquier origen (útil para pruebas locales)
var actualizador = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// -------------------- Tipo de mensaje simplificado --------------------

// MensajeWS representa cualquier mensaje enviado al cliente vía WebSocket
type MensajeWS struct {
	Tipo   string `json:"tipo"`             // "registro", "resumen", "finalizado"
	Topico string `json:"topico,omitempty"` // "mpi" o "openmp"
	Texto  string `json:"texto,omitempty"`  // texto del mensaje
}

// -------------------- MPI (anillo de sectores) --------------------
// correrMPI simula un auto pasando por sectores de manera secuencial
func correrMPI(sectores int, vueltas int, enviar chan MensajeWS) {
	if sectores < 1 {
		enviar <- MensajeWS{Tipo: "registro", Topico: "mpi", Texto: "Error: sectores debe ser >= 1"}
		enviar <- MensajeWS{Tipo: "finalizado", Topico: "mpi"}
		return
	}
	if vueltas < 1 {
		vueltas = 1
	}

	enviar <- MensajeWS{
		Tipo:   "registro",
		Topico: "mpi",
		Texto:  fmt.Sprintf("Iniciando MPI: %d sectores, %d vueltas", sectores, vueltas),
	}

	for v := 1; v <= vueltas; v++ {
		enviar <- MensajeWS{Tipo: "registro", Topico: "mpi", Texto: fmt.Sprintf("=== Vuelta %d ===", v)}

		for s := 1; s <= sectores; s++ {
			tiempo := float64(rand.Intn(2300)+1200) / 100.0 // tiempo aleatorio entre 12.00 y 35.99 s
			enviar <- MensajeWS{
				Tipo:   "registro",
				Topico: "mpi",
				Texto:  fmt.Sprintf("Sector %d recibió tiempo %.2f s (vuelta %d)", s, tiempo, v),
			}
			time.Sleep(300 * time.Millisecond) // simulación de paso por sector
		}
	}

	enviar <- MensajeWS{Tipo: "finalizado", Topico: "mpi", Texto: "MPI finalizado"}
}

// -------------------- OpenMP (vueltas rápidas) --------------------

// ResultadoOpenMP guarda la mejor vuelta de un auto
type ResultadoOpenMP struct {
	AutoID          int
	MejorVuelta     float64
	CantidadVueltas int
}

// correrOpenMP simula varios autos corriendo vueltas rápidas en paralelo usando mutex
func correrOpenMP(cantidadAutos int, vueltas int, enviar chan MensajeWS) {
	if cantidadAutos < 1 {
		enviar <- MensajeWS{Tipo: "registro", Topico: "openmp", Texto: "Error: cantidad de autos debe ser >= 1"}
		enviar <- MensajeWS{Tipo: "finalizado", Topico: "openmp"}
		return
	}
	if vueltas < 1 {
		vueltas = 1
	}

	enviar <- MensajeWS{Tipo: "registro", Topico: "openmp", Texto: fmt.Sprintf("Iniciando OpenMP: %d autos, %d vueltas cada uno", cantidadAutos, vueltas)}

	resultados := make([]ResultadoOpenMP, cantidadAutos)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for auto := 0; auto < cantidadAutos; auto++ {
		wg.Add(1)
		go func(autoID int) {
			defer wg.Done()
			mejor := 1e9
			for v := 1; v <= vueltas; v++ {
				tiempoVuelta := float64(rand.Intn(2099)+7500) / 100.0
				time.Sleep(200 * time.Millisecond)
				enviar <- MensajeWS{Tipo: "registro", Topico: "openmp", Texto: fmt.Sprintf("Auto %d - Vuelta %d: %.2f s", autoID+1, v, tiempoVuelta)}
				if tiempoVuelta < mejor {
					mejor = tiempoVuelta
					enviar <- MensajeWS{Tipo: "registro", Topico: "openmp", Texto: fmt.Sprintf("Auto %d - Nueva mejor vuelta: %.2f s", autoID+1, mejor)}
				}
			}
			// Mutex para proteger escritura en slice compartido
			mutex.Lock()
			resultados[autoID] = ResultadoOpenMP{AutoID: autoID + 1, MejorVuelta: mejor, CantidadVueltas: vueltas}
			mutex.Unlock()
		}(auto)
	}

	wg.Wait()

	// Calcula mejor vuelta general
	mejorGeneral := ResultadoOpenMP{AutoID: -1, MejorVuelta: 1e9}
	mutex.Lock()
	for _, r := range resultados {
		if r.MejorVuelta < mejorGeneral.MejorVuelta {
			mejorGeneral = r
		}
	}
	mutex.Unlock()

	enviar <- MensajeWS{Tipo: "resumen", Topico: "openmp", Texto: fmt.Sprintf("Resultados OpenMP:\nMejor por auto: %+v\nMejor general: %+v", resultados, mejorGeneral)}
	//enviar <- MensajeWS{Tipo: "resumen", Topico: "mpi", Texto: "OpenMP finalizado"}
	enviar <- MensajeWS{Tipo: "finalizado", Topico: "openmp", Texto: "OpenMP finalizado"}
}

// -------------------- WebSocket handler --------------------

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := actualizador.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error al actualizar a websocket:", err)
		return
	}
	defer conn.Close()

	enviar := make(chan MensajeWS, 100)
	defer close(enviar)

	// Goroutine que envía mensajes de forma segura
	go func() {
		for msg := range enviar {
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("Error escribiendo en websocket:", err)
				return
			}
		}
	}()

	// Bucle principal de lectura de comandos
	for {
		var comando map[string]any
		if err := conn.ReadJSON(&comando); err != nil {
			log.Println("Conexión cerrada o error de lectura:", err)
			return
		}
		switch comando["action"] {
		case "iniciar_mpi":
			sectores := 5
			vueltas := 3
			if v, ok := comando["sectores"].(float64); ok {
				sectores = int(v)
			}
			if v, ok := comando["vueltas"].(float64); ok {
				vueltas = int(v)
			}
			go correrMPI(sectores, vueltas, enviar)
		case "iniciar_openmp":
			autos := 4
			vueltas := 5
			if v, ok := comando["autos"].(float64); ok {
				autos = int(v)
			}
			if v, ok := comando["vueltas"].(float64); ok {
				vueltas = int(v)
			}
			go correrOpenMP(autos, vueltas, enviar)
		default:
			enviar <- MensajeWS{Tipo: "registro", Texto: fmt.Sprintf("Comando no reconocido: %v", comando["action"])}
		}
	}
}

// -------------------- HTTP handler --------------------

var plantillaIndex = template.Must(template.New("index").Parse(htmlIndex))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	plantillaIndex.Execute(w, nil)
}

// -------------------- Main --------------------

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ws", wsHandler)

	addr := ":8080"
	fmt.Println("Servidor corriendo en http://localhost" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// -------------------- HTML + JS embebido --------------------

const htmlIndex = `<!doctype html>
<html>
<head>
<meta charset="utf-8"/>
<title>Simulaciones MPI / OpenMP - Fórmula1</title>
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
<h2>Simulaciones Fórmula1 — MPI (sectores) y OpenMP (vueltas rápidas)</h2>
<div style="display:flex; gap: 16px;">
  <div class="col">
    <h3>MPI - Sectores (anillo)</h3>
    <label>Cantidad de sectores: <input id="mpi-sectores" type="number" value="5" min="1"></label><br>
    <label>Vueltas: <input id="mpi-vueltas" type="number" value="3" min="1"></label><br>
    <button id="start-mpi">Iniciar MPI</button>
    <div style="margin-top:10px;">
      <h4>Salida MPI</h4>
      <div id="mpi-log" class="log-mpi"></div>
    </div>
  </div>

  <div class="col">
    <h3>OpenMP - Vueltas rápidas</h3>
    <label>Autos: <input id="openmp-autos" type="number" value="4" min="1"></label><br>
    <label>Vueltas por auto: <input id="openmp-vueltas" type="number" value="5" min="1"></label><br>
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

ws.onopen = () => appendAmbos("Conexión WebSocket establecida.");
ws.onclose = () => appendAmbos("WebSocket cerrado.");
ws.onerror = (e) => appendAmbos("Error WebSocket: " + e);

ws.onmessage = (evt) => {
  try {
    const msg = JSON.parse(evt.data);
    if(msg.topico==="mpi") append(mpiLog, msg.texto);
    else if(msg.topico==="openmp") append(openmpLog, msg.texto);
    else appendAmbos(msg.texto);
  } catch(e){
    appendAmbos("Mensaje no JSON: "+evt.data);
  }
};

function append(target,text){ const p=document.createElement("div"); p.innerHTML=text; target.appendChild(p); target.scrollTop=target.scrollHeight;}
function appendAmbos(text){ append(mpiLog,text); append(openmpLog,text);}

document.getElementById("start-mpi").onclick = ()=>{
  const sectores=parseInt(document.getElementById("mpi-sectores").value)||5;
  const vueltas=parseInt(document.getElementById("mpi-vueltas").value)||3;
  ws.send(JSON.stringify({action:"iniciar_mpi",sectores:sectores,vueltas:vueltas}));
  append(mpiLog,"<b>Comando enviado: iniciar MPI</b>");
};

document.getElementById("start-openmp").onclick = ()=>{
  const autos=parseInt(document.getElementById("openmp-autos").value)||4;
  const vueltas=parseInt(document.getElementById("openmp-vueltas").value)||5;
  ws.send(JSON.stringify({action:"iniciar_openmp",autos:autos,vueltas:vueltas}));
  append(openmpLog,"<b>Comando enviado: iniciar OpenMP</b>");
};
</script>
</body>
</html>
`
