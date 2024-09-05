# Sincronización

Para la concurrencia decidí utilizar un simple threadpool que arme yo mismo utilizando la libreria threading.

Y para la sincronización utilice los Locks del módulo threading también

Solo necesite dos locks:
* Un lock para el **estado del servidor**. Ahi guardo información sobre cuales son las agencias que ya terminaron de enviar sus apuestas. Es importante que esten sincronizados y dos threads no escriban a la vez, porque sino se podria entrar en un deadlock donde se esta esperando a una agencia que ya reportó que terminó.
* Un lock para que los threads no llamen a la funcion **store_bets** a la vez, ya que esta escribiendo en un archivo y no se puede escribir a la vez.

No hizo falta crear un lock para la funcion load_bets ya que esta solo lee el archivo y esto puede ser hecho por varios threads a la vez sin problemas. Ya que por como funciona el flujo del protocolo, no puede pasar que haya un thread escribiendo mientras otro esta leyendo.
