from unittest.mock import Mock
from protocol import recv_bet

mock_socket = Mock()
mock_socket.recv.side_effect = [b'B', # no cuentan
				      int.to_bytes(24, 4, byteorder='big'), # no cuentan
				      b'A',								   # 1 byte
				      int.to_bytes(4, 4, byteorder='big'), # 4 bytes
				      int.to_bytes(1, 4, byteorder='big'), # 4 bytes
				      b'H',								   # 1 byte
				      int.to_bytes(10, 4, byteorder='big'),# 4 bytes
				      b'2019',						   # 4 bytes
				      b'-12-04',						   # 6 bytes
				      ]

print(recv_bet(mock_socket))