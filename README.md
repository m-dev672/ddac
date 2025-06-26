<br />
  <p align="center">
    <img src="images/logo.svg" alt="Logo" width="80" height="80">
  </p>

  <h3 align="center">DDUC</h3>

  <p align="center">
    Decentralized Database on Universal Chain
  </p>
</div>

## About The Project

The project is a decentralized network of SQL nodes (airports) managed via Ethereum smart contracts. SQL nodes are dynamically selected as the origin of the route, receive SQL queries (flight plans), execute them on the SQLite database (destination), and return hash results.

### 1. `contracts/Dispatcher.sol` (Solidity)

The Ethereum smart contract that handles:

- **Airport/Node Management**: Register (`constructAirport`), open (`openAirport`), temporarily close (`closeAirport`), or remove (`destroyAirport`) an airport/node.
- **Route/Database Management**: Launch (`launchNewRoute`), modify (`changePermission`), terminate (`terminateRoute`), or reroute (`reroute`) routes to destination/database registration.
- **Query/FlightPlan Submit**: Submit SQL queries/flightplan (`submitFlightPlan`) to be executed by nodes, databases on the route.


### 2. `src/server.go` (Golang)

The Go-based node implementation that:

- Registers itself as an airport on Ethereum.
- Listens for `NewRouteLaunched` and `RouteTerminated` to manage databases.
- Listens for `FlightPlanSubmitted` events.
- Parses and executes SQL queries.
- Calculates hash of results for data changes.


## ⚙️ Requirements
- [Hardhat](https://hardhat.org/)
- [Go 1.24.4+](https://go.dev/)
- Dependencies:
  ```bash
  go get github.com/google/uuid
  go get github.com/ethereum/go-ethereum
  go get github.com/mattn/go-sqlite3
  go get github.com/rqlite/sql
  ```
