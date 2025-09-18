# Simulaciones de Fórmula 1 con MPI y OpenMP (Didáctico)

## Integrantes

- Marcos Natanael Da Silva
- Exequiel Andres Diaz
- Joaquín Küster
- Lázaro Ezequiel Martinez

## Introducción

Este proyecto implementa ejercicios didácticos de **MPI** y **OpenMP** en el lenguaje Go, para la cátedra de Paradigmas y Lenguajes de Programación de la Universidad de Misiones, adaptados al contexto de carreras de Fórmula 1.  
A través de un servidor HTTP con WebSocket, se puede simular en tiempo real:

- **MPI (Message Passing Interface):** un auto recorriendo sectores de la pista.
- **OpenMP (Open Multi-Processing):** varios autos compitiendo para lograr la vuelta más rápida.

Los resultados se muestran en una interfaz HTML interactiva que permite parametrizar las simulaciones y visualizar los resultados en tiempo real.

---

## 1. ¿Cumplen los ejercicios con lo que establece MPI y OpenMP?

- **MPI:** El ejercicio simula un anillo de nodos (sectores de pista), donde el mensaje (tiempo de paso) viaja entre los sectores. Esto respeta el modelo de paso de mensajes de MPI.
- **OpenMP:** El ejercicio simula múltiples autos (nodos) ejecutando vueltas rápidas en paralelo, lo que representa el modelo de ejecución compartida y paralela de OpenMP.

En ambos casos, se respeta la esencia de cada modelo, aunque simplificado para fines didácticos.

---

## 2. ¿Manejan paralelización? ¿Cómo?

**Sí ✅.**

En Go, la paralelización se logra a través de goroutines.

- En el caso de MPI, cada sector funciona como una goroutine que pasa mensajes al siguiente.
- En OpenMP, cada auto es una goroutine que ejecuta vueltas en paralelo y reporta su tiempo de vuelta.

Además, se utilizan canales (`channels`) para sincronización y comunicación, imitando los mecanismos de MPI y OpenMP.

---

## 3. ¿Son paralelos y concurrentes?

- **Concurrentes:** Siempre, porque las goroutines permiten que múltiples tareas estén en progreso al mismo tiempo.
- **Paralelos:** Solo si existen varios cores disponibles y el runtime de Go los utiliza.

Para garantizarlo, se puede configurar:

```go
import "runtime"
func init() {
    runtime.GOMAXPROCS(runtime.NumCPU()) // usar todos los cores
}
```

Entonces:

- **Concurrencia:** ✅ asegurada.
- **Paralelismo:** ✅ depende del hardware (multicore) y configuración del runtime.

---

## 4. Explicación de los ejercicios didácticos

### MPI – Sectores en anillo

- Los sectores de la pista son nodos de un anillo.
- Un auto simulado pasa por cada sector, generando un tiempo aleatorio (12 a 35 segundos).
- Se envía un mensaje en tiempo real al cliente con el formato:  
  `Tiempo de sector X: Y segundos (vuelta Z).`

### OpenMP – Vueltas rápidas

- Cada auto es un nodo independiente.
- Se simula que cada auto corre un número de vueltas (con tiempos entre 75 y 95 segundos).
- Cada goroutine (auto) informa su tiempo de vuelta y si logró una mejor vuelta personal.
- Al finalizar, se calcula el mejor tiempo general.

---

## 5. Estructura del proyecto

```
.
├── main.go        # Servidor HTTP y WebSocket
├── go.mod         # Módulo de Go
└── README.md      # Documentación (este archivo)
```

Dentro de `main.go`:

- `runMPI()`: Lógica de la simulación MPI.
- `runOpenMP()`: Lógica de la simulación OpenMP.
- `wsHandler()`: Manejo de WebSockets para enviar resultados en tiempo real.
- `indexHTML`: Interfaz HTML embebida con formularios para parametrizar y mostrar resultados.

---

## 6. Cómo ejecutar el proyecto

Ahora se recomienda ejecutar el proyecto usando **Docker Compose**, lo que facilita levantar el servidor sin instalar Go localmente.

### 6.1. Clonar el repositorio

```bash
git clone https://github.com/joaquinkuster/MPI-OpenMP.git
cd MPI-OpenMP
```

### 6.2. Levantar el proyecto con Docker Compose

Desde la raíz del repositorio, ejecutar:

```bash
docker-compose up --build -d
```

**Explicación:**

- `up`: levanta los servicios definidos en `docker-compose.yml`.
- `--build`: construye la imagen antes de ejecutar el contenedor.
- `-d`: ejecuta los contenedores en segundo plano (detached mode).

### 6.3. Acceder a la interfaz web

Después de levantar el contenedor, abrir en el navegador:  
[http://localhost:8080](http://localhost:8080)

La interfaz permitirá:

- **MPI (sectores):** ingresar número de sectores y vueltas.
- **OpenMP (autos):** ingresar número de autos y vueltas por auto.

Los resultados se mostrarán en tiempo real gracias a WebSockets.

---

## 7. Conclusiones

El proyecto muestra de manera didáctica y simplificada los conceptos de MPI y OpenMP aplicados a un escenario de Fórmula 1.  
Se logra concurrencia garantizada y paralelismo potencial en hardware multinúcleo.

La estructura permite extender el proyecto para:

- Visualizaciones gráficas más avanzadas.
- Comparación de simulaciones MPI vs OpenMP en métricas reales.
- Persistencia de resultados para análisis posterior.

