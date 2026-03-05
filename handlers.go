// Archivo handlers.go
package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Solicitud HTTP simplificada
// - Method: método HTTP
// - RawPath: path original del request
// - Route: path sin query string
// - Query: parámetros parseados
// - Body: contenido del request
type Request struct {
	Method  string
	RawPath string
	Route   string
	Query   url.Values
	Body    string
}

func get(conn net.Conn, db *sql.DB) {
	defer conn.Close()

	// Parseo de la solicitud HTTP de forma manual.
	readR, ok := readReq(conn)
	if !ok {
		return
	}

	switch readR.Route {

	case "/":
		if readR.Method != "GET" {
			write_Text(conn, 405, "Method Not Allowed")
			return
		}
		handleHome(conn, db)
		return

	case "/create":
		if readR.Method == "GET" {
			handleCreateForm(conn)
			return
		}
		if readR.Method == "POST" {
			handleCreateSubmit(conn, db, readR.Body)
			return
		}
		write_Text(conn, 405, "Method Not Allowed")
		return

	// /update modifica datos: se asigna a la funcion POST.
	case "/update":
		if readR.Method != "POST" {
			write_Text(conn, 405, "Method Not Allowed")
			return
		}
		handleUpdate(conn, db, readR.Query)
		return

	// /delete elimina datos: se asigna a la funcion DELETE.
	case "/delete":
		if readR.Method != "DELETE" {
			write_Text(conn, 405, "Method Not Allowed")
			return
		}
		handleDelete(conn, db, readR.Query)
		return

	// Ruta del archivo JS y CSS para la parte frontend.
	case "/Complemento.js":
		serve_static(conn, "Complemento.js", "application/javascript")
		return

	case "/styles.css":
		serve_static(conn, "styles.css", "text/css")
		return

	default:
		write_HTML(conn, 404, "<html><body><h1>404 Not Found</h1></body></html>")
		return
	}
}

// Redireccionamiento a otra ruta, con el código HTTP 303 See Other.
func readReq(conn net.Conn) (Request, bool) {
	r := bufio.NewReader(conn)

	// Lectura del request line.
	readReqLine, err := r.ReadString('\n')
	if err != nil {
		return Request{}, false
	}
	readReqLine = strings.TrimRight(readReqLine, "\r\n")
	parts := strings.Fields(readReqLine)
	if len(parts) < 2 {
		return Request{}, false
	}

	method := parts[0]
	rawPath := parts[1]

	//Leectura de los headers y deteccion  del Content Length
	contentLength := 0
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return Request{}, false
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(k))
		val := strings.TrimSpace(v)
		if key == "content-length" {
			n, _ := strconv.Atoi(val)
			contentLength = n
		}
	}

	body := ""
	if contentLength > 0 {
		buf := make([]byte, contentLength)
		if _, err := io.ReadFull(r, buf); err != nil {
			return Request{}, false
		}
		body = string(buf)
	}

	route, queryStr, _ := strings.Cut(rawPath, "?")
	q, _ := url.ParseQuery(queryStr)

	// Separacion de la ruta y query string
	return Request{
		Method:  method,
		RawPath: rawPath,
		Route:   route,
		Query:   q,
		Body:    body,
	}, true
}

// handleHome genera HTML SSR para listar series.
func handleHome(conn net.Conn, db *sql.DB) {
	rows, err := db.Query("SELECT id, name, current_episode, total_episodes FROM series ORDER BY id ASC")
	if err != nil {
		body := fmt.Sprintf("<html><body><h1>Error DB</h1><pre>%s</pre></body></html>", htmlEscape(err.Error()))
		write_HTML(conn, 500, body)
		return
	}
	defer rows.Close()

	var sb strings.Builder
	for rows.Next() {
		var id int
		var name string
		var current int
		var total int
		_ = rows.Scan(&id, &name, &current, &total)

		sb.WriteString(fmt.Sprintf(
			`<tr>
				<td>%d</td>
				<td>%s</td>
				<td>%d</td>
				<td>%d</td>
				<td>
					<button onclick="prevEpisode(%d)">-1</button>
					<button onclick="nextEpisode(%d)">+1</button>
					<button onclick="deleteSeries(%d)">Eliminar</button>
				</td>
			</tr>`+"\n",
			id, htmlEscape(name), current, total, id, id, id,
		))
	}

	tpl, err := load_Template("index.html")
	if err != nil {
		body := fmt.Sprintf("<html><body><h1>Error</h1><pre>No se pudo leer el archivo index.html: %s</pre></body></html>", htmlEscape(err.Error()))
		write_HTML(conn, 500, body)
		return
	}

	//Se sustituye el placeholder {{ROWS}} por el HTML generado
	page := strings.Replace(tpl, "{{ROWS}}", sb.String(), 1)
	write_HTML(conn, 200, page)
}

// Respuesta al formulario HTML para crear una serie.
func handleCreateForm(conn net.Conn) {
	tpl, err := load_Template("create.html")
	if err != nil {
		body := fmt.Sprintf("<html><body><h1>Error</h1><pre>No se pudo leer el archivo create.html: %s</pre></body></html>", htmlEscape(err.Error()))
		write_HTML(conn, 500, body)
		return
	}
	write_HTML(conn, 200, tpl)
}

// Validacion de los datos ingresados para su posterior insercion en SQL.
func handleCreateSubmit(conn net.Conn, db *sql.DB, body string) {
	values, err := url.ParseQuery(body)
	if err != nil {
		write_Text(conn, 400, "Bad form")
		return
	}

	name := strings.TrimSpace(values.Get("series_name"))
	current_episode_str := values.Get("current_episode")
	total_episode_str := values.Get("total_episodes")

	current, err1 := strconv.Atoi(current_episode_str)
	total, err2 := strconv.Atoi(total_episode_str)

	// Validación de parametros para que no sean nulos.
	if name == "" || err1 != nil || err2 != nil || current < 1 || total < 1 || current > total {
		write_Text(conn, 400, "Invalid values")
		return
	}

	// Placeholders (?) para evitar SQL injection
	_, err = db.Exec(
		"INSERT INTO series (name, current_episode, total_episodes) VALUES (?, ?, ?)",
		name, current, total,
	)
	if err != nil {
		write_Text(conn, 500, "DB insert error")
		return
	}

	write_Redirect_303(conn, "/")
}

// Actualizacion de la propiedad current_episode mediante deltas/cantidades(+1 o -1).
func handleUpdate(conn net.Conn, db *sql.DB, q url.Values) {
	id_String := q.Get("id")
	id, err := strconv.Atoi(id_String)
	if err != nil || id <= 0 {
		write_Text(conn, 400, "Bad id")
		return
	}

	delta, err := strconv.Atoi(q.Get("delta"))
	if err != nil || (delta != 1 && delta != -1) {
		write_Text(conn, 400, "Bad delta")
		return
	}

	if delta == 1 {
		_, err = db.Exec(`UPDATE series SET current_episode = current_episode + 1 WHERE id = ? AND current_episode < total_episodes`, id)
	} else {
		_, err = db.Exec(`UPDATE series SET current_episode = current_episode - 1 WHERE id = ? AND current_episode > 1`, id)
	}

	if err != nil {
		write_Text(conn, 500, "DB update error")
		return
	}

	write_Text(conn, 200, "ok")
}

// Eliminacion de una serie mediante su id.
func handleDelete(conn net.Conn, db *sql.DB, q url.Values) {
	idStr := q.Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		write_Text(conn, 400, "Bad id")
		return
	}

	_, err = db.Exec("DELETE FROM series WHERE id = ?", id)
	if err != nil {
		write_Text(conn, 500, "DB delete error")
		return
	}

	write_No_Content(conn)
}
