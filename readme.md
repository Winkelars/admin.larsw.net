# Plan Umstieg auf Dockercontainer

- Ein Container für Frontend, ein Container für Backend
- Frontend-Container:
  - Dockerfile wird als Basis ein Alpine-Nginx-Image nehmen
  - Build-Befehle:
    - `cd /home/l/vps-admin-gui/apps/web/`
    - `npm build` bzw. `npx vite build`
    - Inhalt von dist danach in den Container kopieren lassen

- Backend-Container:
  - Golang-Image
  - Build-Befehle:
    - `cd /home/l/vps-admin-gui/apps/api/`
    - `go build .` (glaube so war der Befehl)
    - die Binary in den Container kopieren lassen (Port in Env-Variablen?)

- Dockerfiles in einer Docker-Compose-Datei referenzieren
  - Im Sourcecode die Ports via Env-Variablen definieren

# Überlegungen zur API

- Ich will eigentlich, dass API Calls nicht aus dem Internet, sondern von 127.0.0.1 aus verschickt werden
- Vite stellt lediglich Frontends zur Verfügung und gibt dem Client eventuell keine Möglichkeit API-Calls für den Server zu kommandieren
  - => Quick LLM-Research:
    Ja, Vite (oder vielmehr die via Vite erzeugte SPA) kann das auf keinen Fall
    Aber man kann die SPA einfach in das Go-Backend "einbetten" anstatt Nginx oder Apache zu benutzen - Die Webpages werden dann halt zusätzlichen HTML-Endpunkten, neben den JSON-Endpunkten

    Das heißt dann BFF - Backend for Frontend - und ich hab es schonmal mit Python FastAPI und HTMX gemacht ohne den Namen zu kennen

    API-Calls werden dann immer noch aus dem Internet verschickt - Aber das ist sowieso via SPA unvermeidbar wenn man DOM-Änderungen will
    Mit BFF läuft alles über den selben Port
    Eine Bibliothek in Go kann einfach den "dist"-Ordner, den Vite baut, nehmen und in die Go-Binary einbetten
    Dann kann das ganze Projekt in einer leicht zu distributierenden Datei gespeichert werden _(was eigentlich unnötig ist, da das Projekt extrem spezifische Anforderungen löst und außer mir von niemandem gebraucht wird)_

    ...aber zumindest kann ich eine semi-professionelle CI/CD-Pipeline machen

# Wir halten fest: Wenn Lars eine Notiz anfängt, ist die ursprüngliche Überschrift selten lange von Bedeutung

## Die Binary kann garantiert mit einem fertigen Go-Image kombiniert werden
