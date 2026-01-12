`
# chili-traductor-go 🌶️

chili-translator-go es un contenedor de traducción automática universal escrito en Go. Está diseñado para traducir scripts (.sh, .py), archivos de documentación (Markdown) y archivos de datos (JSON) manteniendo la integridad de variables, enlaces y sintaxis técnica.

Su principal ventaja es Smart Cache v2.1.9, que reduce drásticamente las llamadas de red y acelera las traducciones repetitivas mediante la reutilización de datos locales.

## ✨ Características

* Multiformato: Soporta .sh, .py, .md, .json, .yaml.
* Preservación de sintaxis: protege automáticamente las variables de shell ($VAR, ${VAR}), enlaces de Markdown y marcadores de posición de cadenas durante el proceso de traducción.
* Traducción paralela: Procesa múltiples idiomas simultáneamente usando Goroutines (ajustable vía -j).
* Caché persistente con marca de tiempo: almacena las traducciones localmente y gestiona el ciclo de vida de los datos, lo que permite una limpieza inteligente.
* Interfaz progresiva: visualización en tiempo real del progreso de cada idioma con una alineación visual perfecta, independientemente del tamaño del código de idioma (por ejemplo, en vs zh-CN).

## 🚀 Instalación

Asegúrate de tener Go instalado y las dependencias del sistema (gettext, trans).
```bash
git clon https://github.com/chililinux/chili-tradutor-go.git
cd chili-traductor-go/src
ir a construir -o chili-translator-go chili-translator-go-v2.1.9.go
sudo mv chili-translator-go /usr/local/bin/
```

## 🛠️ Uso

### Traducción básica
Para traducir un archivo a idiomas estándar (EN, ES, IT, DE, FR, RU, ZH, JA, KO):

chile-traductor-go -i meu_script.sh


### Especificación de idiomas y motor

cheli-treducer-go -y tutorial.md


### Borrado de caché
Elimine las entradas de caché que no se hayan utilizado en los últimos 30 días:

traductor-chili-go --clean-cache


## ⚙️ Opciones (Banderas)

| Bandera | Largo | Descripción |
| :--- | :--- | :--- |
| -yo | --archivo de entrada | Archivo fuente para la traducción. |
| -e | --motor | Motor de traducción: Google, Bing, Yandex (predeterminado: Google). |
| -s | --fuente | Idioma de origen (por ejemplo: pt, en) (predeterminado: automático). |
| -l | --idioma | Lista de idiomas separados por coma o todos. |
| -j | --trabajos | Número de traducciones simultáneas (predeterminado: 8). |
| -f | --fuerza | Fuerza la traducción omitiendo el caché local. |
| | --clean-cache | Elimina elementos de caché obsoletos (más de 30 días). |
| -q | --tranquilo | Modo silencioso (sin progreso visual). |
| -v | --detallado | Muestra detalles técnicos mientras se ejecuta. |
| -V | --versión | Muestra la versión actual. |

## 📁 Estructura de salida

* Scripts/POT: Genera archivos .po en ./pot/ y archivos binarios .mo en ./usr/share/locale/.
* Markdown: Genera versiones traducidas en ./doc/ (ej: README-en.md).
* JSON: Genera versiones traducidas en ./translated/.

## 🛡️ Lógica de caché (v2.1.9)

El caché se almacena en ~/.cache/chili-tradutor-go/cache.json.

* Migración automática: al detectar registros de versiones anteriores (v2.1.8), la herramienta imprime automáticamente la marca de tiempo actual en registros heredados para evitar la pérdida de datos históricos.
* Actualización automática: cada vez que se encuentra un elemento en el caché, se actualiza su marca de tiempo de "Último uso", lo que lo protege de una futura limpieza automática.
* Seguridad: la limpieza mediante --clean-cache solo elimina lo que realmente no se utiliza, lo que garantiza que su base de conocimientos de traducción crezca de manera saludable.


Desarrollado por: Vilmar Catafesta <vcatafesta@gmail.com>
Copyright © 2023-2026 Equipo ChiliLinux
