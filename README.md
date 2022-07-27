# CUDOS Stats Service

## Starting the service:

Edit ```config.yaml``` to set rpc and grpc node endpoints, the default values are for mainnet.

Build the docker image:\
```docker build -t 'cudos-stats-v2-service' .```

Run the docker image:\
```docker run -d --name cudos-stats-v2-service -p 3001:3000 cudos-stats-v2-service```

## Available endpoints:

### For Cosmos networks explorers who look for default mint and bank module endpoints:
http://127.0.0.1:3001/cosmos/mint/v1beta1/params\
http://127.0.0.1:3001/cosmos/mint/v1beta1/annual_provisions\
http://127.0.0.1:3001/cosmos/mint/v1beta1/inflation\
http://127.0.0.1:3001/cosmos/bank/v1beta1/supply

### For coinmarketcap and other similar integrations:
http://127.0.0.1:3001/circulating-supply - coinmarketcap endpoint that is returning current circulating supply as decimal.\
http://127.0.0.1:3001/json/circulating-supply - endpoint that is returning current circulating supply as json.