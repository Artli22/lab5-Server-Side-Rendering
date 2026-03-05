// archivo helpers
package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Lectura de un archivo completo y retorno del contenido de dicho archivo como string.
func load_Template(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Evita que se rompa el flujo de la aplicación al escribir caracteres especiales en el HTML.
func htmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&#39;",
	)
	return r.Replace(s)
}

// Escritura de una respuesta HTTP con Content-Type HTML.
func write_HTML(conn net.Conn, status int, body string) {
	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		status, status_Text(status),
		len([]byte(body)),
		body,
	)
	conn.Write([]byte(response))
}

// Escritura de una respuesta HTTP con Content-Type text/plain.
func write_Text(conn net.Conn, status int, body string) {
	response := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Type: text/plain; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		status, status_Text(status),
		len([]byte(body)),
		body,
	)
	conn.Write([]byte(response))
}

// Redireccionamiento a otra ruta, con el código HTTP 303 See Other.
func write_Redirect_303(conn net.Conn, location string) {
	response := fmt.Sprintf(
		"HTTP/1.1 303 See Other\r\n"+
			"Location: %s\r\n"+
			"Content-Length: 0\r\n"+
			"Connection: close\r\n"+
			"\r\n",
		location,
	)
	conn.Write([]byte(response))
}

// Respuesta 204 No Content, para indicar que no hay contenido que enviar.
func write_No_Content(conn net.Conn) {
	response := "" +
		"HTTP/1.1 204 No Content\r\n" +
		"Content-Length: 0\r\n" +
		"Connection: close\r\n" +
		"\r\n"
	conn.Write([]byte(response))
}

// Sirve archivos de naturaleza estatica, como CSS, JS o imagenes.
func serve_static(conn net.Conn, path string, contentType string) {
	b, err := os.ReadFile(path)
	if err != nil {
		write_Text(conn, 404, "Not found")
		return
	}

	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: %s\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n",
		contentType,
		len(b),
	)

	conn.Write([]byte(response))
	conn.Write(b)
}

// Traduccion de los códigos HTTP a su texto estándar.
func status_Text(code int) string {
	switch code {
	case 200:
		return "OK"
	case 204:
		return "No Content"
	case 303:
		return "See Other"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	default:
		return "OK"
	}
}
