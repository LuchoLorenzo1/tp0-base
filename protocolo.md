# Protocolo

El protocolo es muy simple. Tiene 3 modos para las 3 fases del programa:
1. cuando el cliente manda las apuestas con la informacion de cada persona para ser guardada en el servidor
2. cuando el cliente notifica que no tiene mas apuestas para mandar
3. cuando el cliente pide el resultado con los ganadores de la loteria

Por lo que lo primero que manda el cliente es un único byte que indica el tipo de mensaje para saber en que fase se encuentra.

Lo siguiente que manda en todos los casos son 4 bytes (en big endian) que indican la agencia a la cual pertenece el cliente haciendo la peticion.

## Fase 1: Mandar apuestas

Si estamos en la fase de mandar apuestas (mode == 1), el cliente manda todas sus apuestas al servidor.

Primero manda la cantidad de personas a enviar como un entero de 4 bytes (uint32, big endian), y luego mando cada persona de la forma descripta a continuacion:
* para nombre y apellido, ya que son strings de longitud variable, se envía primero la longitud del string como un entero de tamaño fijo (1 byte basta, ya que el tamaño maximo de ambos es 23 y 10 respectivamente) y luego el string en sí como una cadena de bytes.
* el dni y nacimiento son strings de longitud fija, por lo que se envían directamente como cadenas de bytes.
	* por mas que el dni sea un número, no me parecio adecuado enviarlo como tal, ya que en algun otro caso (como de otro pais) puede llegar a contener letras
* el numero de la loteria si lo mande como un entero de 8 bytes (uint64), y el mismo es encodeado en big endian (el byte mas significativo primero).

El protocolo es dinamico:
* Desventajas: Hay que deserializar persona por persona (de principio a fin, una por una) para saber cuantas personas hay en total. No se puede dividir en chunks y paralelizar la deserilizacion o algo por el estilo.
* Ventajas: se aprovecha mejor el espacio y se mandan menos bytes.

El servidor responde con dos bytes que indican si la operacion fue exitosa. Estos bytes son "OK" (ascii 79 y 75).

## Fase 2: Notificar fin de apuestas

En esta no hay mas que mandar el modo y el numero agencia. Luego el servidor responde con "OK" luego de guardar que esta agencia ya no va a mandar mas apuestas.

## Fase 3: Pedir resultado

En esta fase (modo == 3) el servidor responde con 2 bytes "NO" (ascii) si no esta listo para mandar los resultados, a lo que el cliente tendra que esperar y mandarlo nuevamente luego.

Si el servidor esta listo, responde con "OK" y luego manda los ganadores al cliente.

Para esto manda la cantidad de ganadores en un entero de 4 bytes (uint32, big endian), y luego manda solo el dni de cada ganador (8 bytes).
