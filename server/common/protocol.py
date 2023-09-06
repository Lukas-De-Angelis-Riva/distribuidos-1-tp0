import socket
from utils import Bet

T_LENGTH = 1
L_LENGTH = 4

# Types
BET_TYPE = 'B'
AGENCY_NAME_TYPE = 'A'
NAME_TYPE = 'N'
LAST_NAME_TYPE = 'L'
DOCUMENT_TYPE = 'D'
BIRTHDATE_TYPE = 'H'
NUMBER_TYPE = 'U'

def read_all(socket, bytes_to_read):
    data = b''
    while len(data) < bytes_to_read: 
        try:
            new_data = socket.recv(bytes_to_read - len(data))
        except OSError as e:
            # logging
            raise e
        data += new_data
    return data

def recv_bet(socket):
    """
    Reads from the socket a bet. Using the TLV protocol implemented.
    If something goes wrong, a exception will be arise.
    """

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

    assert raw_bet[AGENCY_NAME_TYPE], "Invalid bet: no agency name provided"
    assert raw_bet[NAME_TYPE], "Invalid bet: no name provided"
    assert raw_bet[LAST_NAME_TYPE], "Invalid bet: no lastname provided"
    assert raw_bet[DOCUMENT_TYPE], "Invalid bet: no document provided"
    assert raw_bet[BIRTHDATE_TYPE], "Invalid bet: no birthdate provided"
    assert raw_bet[NUMBER_TYPE], "Invalid bet: no number provided"

    return Bet(
        agency=str(int.from_bytes(raw_bet[AGENCY_NAME_TYPE], byteorder='big')), 
        first_name=raw_bet[NAME_TYPE].decode('utf-8'),
        last_name=raw_bet[LAST_NAME_TYPE].decode('utf-8'),
        document=raw_bet[DOCUMENT_TYPE].decode('utf-8'),
        birthdate=raw_bet[BIRTHDATE_TYPE].decode('utf-8'),
        number=str(int.from_bytes(raw_bet[NUMBER_TYPE], byteorder='big'))
    )