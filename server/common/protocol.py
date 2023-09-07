import socket
import struct
from common.utils import Bet

T_LENGTH = 1
L_LENGTH = 4

# Data tags
BATCH_TYPE = 'Z'            # Chunk 
BET_TYPE = 'B'              # Apuesta
AGENCY_NAME_TYPE = 'A'      # Agencia
NAME_TYPE = 'N'             # Nombre
LAST_NAME_TYPE = 'L'        # Apellido
DOCUMENT_TYPE = 'D'         # Documento
BIRTHDATE_TYPE = 'H'        # Fecha de nacimiento
NUMBER_TYPE = 'U'           # Numero de apuesta

# server responses types
WINNERS_TYPE = 'W'          # TAG: se envian los ganadores
AWAIT_TYPE = 'Y'            # TAG: aun no se sortearon los ganadores
OK_TYPE = 'O'               # TAG: acknowledge de haber recibido correctamente los datos

# client requests types
POLL_TYPE = 'P'             # TAG: cliente solicita ganadores del sorteo
FINISH_TYPE = 'F'           # TAG: cliente ya no envia mas apuestas

# Requests
UPLOAD_BETS_REQ = 1         # REQUEST de carga de apuestas 
FINISH_REQ = 2              # REQUEST de finalización de comunicación
POLL_WINNERS_REQ = 3        # REQUEST de solicitud de ganadores

def read_all(socket, bytes_to_read):
    """
    Lee del socket la cantidad exacta de bytes.
    En caso de error levanta una excepcion 
    """
    data = b''
    while len(data) < bytes_to_read: 
        try:
            new_data = socket.recv(bytes_to_read - len(data))
        except OSError as e:
            # logging
            raise e
        data += new_data
    return data

def handle_bet(socket, withType=False):
    """
    Lee del socket una apuesta. Usa el protocolo TLV implementado.
    Si algo va mal, levanta una excepcion
    """
    if withType:
        tlv_type = read_all(socket, T_LENGTH) # Deberia recibir el 'B'
        assert tlv_type.decode('utf-8') == BET_TYPE, "Invalid type: BET excepted"

    bet_len_d = read_all(socket, L_LENGTH) # Deberia recibir el largo del BET (sin contar ni el 'B' ni el largo)
    bet_len = int.from_bytes(bet_len_d, byteorder='big')
    assert bet_len >= 0, "Invalid length: must be positive"

    bytes_received = 0
    raw_bet = {
        AGENCY_NAME_TYPE: b'',
        NAME_TYPE: b'',
        LAST_NAME_TYPE: b'',
        DOCUMENT_TYPE: b'',
        BIRTHDATE_TYPE: b'',
        NUMBER_TYPE: b'',
    }

    while bytes_received < bet_len:
        tlv_type_d = read_all(socket, T_LENGTH)
        tlv_type = tlv_type_d.decode('utf-8')
        bytes_received+=T_LENGTH
        assert bytes_received < bet_len, f"Invalid bet length: field {tlv_type} corrupted"

        field_len_d = read_all(socket, L_LENGTH)
        field_len = int.from_bytes(field_len_d, byteorder='big')
        bytes_received+=L_LENGTH
        assert bytes_received < bet_len, f'Invalid bet length: field {tlv_type} no info provided'
        
        field = read_all(socket, field_len)
        bytes_received+=field_len
        assert bytes_received <= bet_len, f'Invalid bet length: more information received'

        # If it already exist, will be overwritted
        raw_bet[tlv_type] = field

    # Verificacion de que la apuesta se haya recibido completamente
    assert raw_bet[AGENCY_NAME_TYPE], "Invalid bet: no agency name provided"
    assert raw_bet[NAME_TYPE], "Invalid bet: no name provided"
    assert raw_bet[LAST_NAME_TYPE], "Invalid bet: no lastname provided"
    assert raw_bet[DOCUMENT_TYPE], "Invalid bet: no document provided"
    assert raw_bet[BIRTHDATE_TYPE], "Invalid bet: no birthdate provided"
    assert raw_bet[NUMBER_TYPE], "Invalid bet: no number provided"

    return Bet(
        agency=raw_bet[AGENCY_NAME_TYPE].decode('utf-8'),
        first_name=raw_bet[NAME_TYPE].decode('utf-8'),
        last_name=raw_bet[LAST_NAME_TYPE].decode('utf-8'),
        document=raw_bet[DOCUMENT_TYPE].decode('utf-8'),
        birthdate=raw_bet[BIRTHDATE_TYPE].decode('utf-8'),
        number=raw_bet[NUMBER_TYPE].decode('utf-8'),
    )

def handle_batch(socket):
    """
    Lee del socket un chunk de apuestas. Usando el protocolo TLV implementado.
    Primero lee el total de apuestas que fueron enviadas.

    Si algo va mal, levanta una excepcion.
    """
    batch_size_d = read_all(socket, L_LENGTH)
    batch_size = int.from_bytes(batch_size_d, byteorder='big')
    assert batch_size >= 0, "Invalid length: must be positive"

    bets = []
    for _ in range(batch_size):
        bets.append(handle_bet(socket, withType=True))

    return bets

# Handles poll request
# ['P' | agency_no:4bytes ]
def handle_poll(socket):
    """
    Lee del socket el numero de la agencia que realiza la solicitud de POLL

    Observacion: no hace falta leer el tlv_type porque fue leido previamente en `recv_req`
    """
    agency_name_d = read_all(socket, L_LENGTH)
    agency_name = int.from_bytes(agency_name_d, byteorder='big')
    return agency_name

def recv_req(socket):
    """
    Lee del socket el primer byte y determina que clase de solicitud es:
        * Cargar una apuesta
        * Cargar multiples apuestas
        * Finalizar la comunicacion
        * Solicitud de ganadores
    Invoca el handler adecuado para la solicitud.
    
    Devuelve el tipo de request y los datos leidos (segun tipo de request)
    
    En caso de algun error se levanta una excepción.
    """
    tlv_type_d = read_all(socket, T_LENGTH)
    tlv_type = tlv_type_d.decode('utf-8')

    if tlv_type == BET_TYPE:
        return UPLOAD_BETS_REQ, [handle_bet(socket, withType=False)]

    elif tlv_type == BATCH_TYPE:
        return UPLOAD_BETS_REQ, handle_batch(socket)

    elif tlv_type == FINISH_TYPE:
        return FINISH_REQ, []

    elif tlv_type == POLL_TYPE:
        return POLL_WINNERS_REQ, handle_poll(socket)

    else:
        raise ValueError("Unknown TYPE")

def write_all(socket, data):
    """
    Escribe sobre socket el total de elementos en data.
    Realiza un loop sobre socket.send para evitar anomalias de short-write.

    En caso de error al enviar los datos, se levanta una excepcion
    """
    bytes_sent = 0
    while bytes_sent < len(data):
        b = socket.send(data[bytes_sent:])
        bytes_sent += b
    return bytes_sent

def confirm_req(socket):
    """
    Envia por el socket el byte 'OK_TYPE' para confirmar la recepción de una apuesta o un chunk
    """
    data = OK_TYPE.encode('utf-8') 
    assert write_all(socket, data) == len(data), "Error in confirmation, cannot write all bytes due to an error"

def force_to_wait(socket):
    """
    Envia por el socket el byte 'AWAIT_TYPE' para indicar al cliente que aun no se realizo el sorteo
    """
    data = AWAIT_TYPE.encode('utf-8')
    assert write_all(socket, data) == len(data), "Error forcing the client to wait, cannot write all bytes due to an error"

def notify_winners(socket, winners):
    """
    Envia por el socket a los ganadores del sorteo almacenados en la lista winners

    Sigue el protocolo TLV, enviando primero el TAG de 'WINNERS_TYPE' y luego los documentos de la siguiente forma:

    WINNERS_TYPE | 3 | [ DOCUMENT_TYPE | 0x1332441 | DOCUMENT_TYPE | 0x23441202 | DOCUMENT_TYPE | 0x21004412 ]

    Siendo 3 la cantidad de documentos a enviar (no los bytes).
    """
    data = b''
    data += WINNERS_TYPE.encode('utf-8')         # Adding tag WINNERS
    data += int.to_bytes(len(winners), 4, 'big') # Adding how many winners

    for winner in winners:
        encoded_document = winner.encode('utf-8')
        data += DOCUMENT_TYPE.encode('utf-8')
        data += int.to_bytes(len(encoded_document), 4, 'big')
        data += encoded_document

    assert write_all(socket, data) == len(data), "Error notifying winners, cannot write all bytes due to an error"