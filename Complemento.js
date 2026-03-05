// Funciones para manejar las acciones de los botones en la interfaz web.

// Función para avanzar al siguiente episodio de una serie.
async function nextEpisode(id) {
  await fetch(`/update?id=${id}&delta=1`, { method: "POST" });
  location.reload();
}

// Función para retroceder al episodio anterior de una serie.
async function prevEpisode(id) {
  await fetch(`/update?id=${id}&delta=-1`, { method: "POST" });
  location.reload();
}

// Función para eliminar una serie de la lista.
async function deleteSeries(id) {
  await fetch(`/delete?id=${id}`, { method: "DELETE" });
  location.reload();
}