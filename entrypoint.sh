#!/usr/bin/env sh

# la ruta ~/.config debe ser montada de un volumen, para que el token
# de Tidal sobreviva. Pero el fichero mopidy.conf también debe estar
# en esa ruta. Lo que hago es que si no existe, lo copio
mkdir ~/.config/mopidy
cp -f  /etc/mopidy.conf ~/.config/mopidy/mopidy.conf

# Y ahora ya puedo lanzar mopidy
exec /usr/bin/mopidy "$@"
