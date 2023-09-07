# pip install yaml
import yaml
import argparse


def info_server(n_clients):
    return {
        'container_name': 'server',
        'image': 'server:latest',
        'entrypoint': 'python3 /main.py',
        'environment': ['PYTHONUNBUFFERED=1',
                        'LOGGING_LEVEL=DEBUG',
                        f'AGENCIES={n_clients}'],
        'volumes': ['./server/config.ini:/config.ini'],
        'networks': ['testing_net'],
    }


def info_client(id):
    return {
        'container_name': f'client{id}',
        'image': 'client:latest',
        'entrypoint': '/client',
        'environment': [
            f'CLI_ID={id}',
            'CLI_LOG_LEVEL=DEBUG',
            f'CLI_BETS_FILE=agency-{id}.csv',
            f'CLI_BETS_BATCH_SIZE={id*50}'
            ],
        'volumes': [
            './client/config.yaml:/config.yaml',
            f'./.data/agency-{id}.csv:/agency-{id}.csv',
        ],
        'networks': ['testing_net'],
        'depends_on': ['server'],
    }


def info_testing_net():
    return {
        'ipam': {
            'driver': 'default',
            'config': [{'subnet': '172.25.125.0/24'}],
        }
    }


def create_file(n_clients):
    config = {}
    config['version'] = '3.9'
    config['name'] = 'tp0'
    config['services'] = {}
    config['services']['server'] = info_server(n_clients)
    for i in range(n_clients):
        config['services'][f'client{i+1}'] = info_client(i+1)
    config['networks'] = {}
    config['networks']['testing_net'] = info_testing_net()

    with open('docker-compose-dev.yaml', 'w') as file:
        yaml.dump(config, file)


if __name__ == '__main__':

    parser = argparse.ArgumentParser()
    parser.add_argument('num_clients', nargs='?', const=1,
                        type=int, help='number of clients', default=1)
    args = parser.parse_args()
    create_file(args.num_clients)
