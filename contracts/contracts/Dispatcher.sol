// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

// Uncomment this line to use console.log
// import "hardhat/console.sol";

contract Dispatcher {
    // About Node
    string[] public airportCodes;

    struct Airport {
        address owner;
        bool available;
        bool exists;
    }
    mapping(string => Airport) public airports;

    function constructAirport(
        string memory airportCode
    ) public {
        if (!airports[airportCode].exists) {
            airportCodes.push(airportCode);

            airports[airportCode] = Airport({
                owner: msg.sender,
                available: true,
                exists: true
            });
        }
    }

    function openAirport(
        string memory airportCode
    ) public {
        if (airports[airportCode].owner == msg.sender && !airports[airportCode].available) {
            airportCodes.push(airportCode);

            airports[airportCode].available = true;
        }
    }

    function closeAirport(
        string memory airportCode
    ) public {
        if (airports[airportCode].owner == msg.sender) {
            for (uint i = 0; i < airportCodes.length; i++) {
                if (keccak256(bytes(airportCodes[i])) == keccak256(bytes(airportCode))) {
                    airportCodes[i] = airportCodes[airportCodes.length - 1];
                    airportCodes.pop();
                    break;
                }
            }

            airports[airportCode].available = false;
        }
    }

    function destroyAirport(
        string memory airportCode
    ) public {
        if (airports[airportCode].owner == msg.sender) {
            closeAirport(airportCode);
            airports[airportCode].exists = false;
        }
    }

    // About Database
    uint seed;
    function pseudoRandom() internal returns (uint) {
        seed++;
        return uint(keccak256(abi.encodePacked(block.prevrandao, block.timestamp, msg.sender, seed)));
    }

    struct Route {
        string[5] origins;
        address[] canAuthorizeAccounts;
        address[] canWriteAccounts;
        string defaultPermission;
        bool exists;
    }
    mapping(string => Route) public routes;

    event NewRoute(string indexed origin, string destination, address[] canWriteAccount, string defaultPermission);
    
    function launchNewRoute(
        string memory destination,
        address[] memory canAuthorizeAccounts,
        address[] memory canWriteAccounts,
        string memory defaultPermission
    ) public {
        if (!routes[destination].exists) {
            require(airportCodes.length >= 5, "Need at least 5 available airports");

            uint n = airportCodes.length;
            string[] memory pool = airportCodes;
            string[5] memory origins;

            for (uint i = 0; i < 5; i++) {
                uint j = i + (pseudoRandom() % (n - i));
                (pool[i], pool[j]) = (pool[j], pool[i]);
                origins[i] = pool[i];
            }

            routes[destination] = Route({
                origins: origins,
                canAuthorizeAccounts: canAuthorizeAccounts,
                canWriteAccounts: canWriteAccounts,
                defaultPermission: defaultPermission,
                exists: true
            });

            for (uint i = 0; i < 5; i++) {
                emit NewRoute(origins[i], destination, canWriteAccounts, defaultPermission);
            }
        }
    }

    function canAuthorize(string memory destination) internal view returns (bool) {
        address[] memory canAuthorizeAccounts = routes[destination].canAuthorizeAccounts;

        for (uint i = 0; i < canAuthorizeAccounts.length; i++) {
            if (canAuthorizeAccounts[i] == msg.sender) {
                return true;
            }
        }

        return false;
    }

    event permissionChanged(string indexed destination, address[] canWriteAccount, string defaultPermission);

    function changePermission(
        string memory destination,
        address[] memory canWriteAccounts,
        string memory defaultPermission
    ) public {
        if (canAuthorize(destination)) {
            routes[destination].canWriteAccounts = canWriteAccounts;
            routes[destination].defaultPermission = defaultPermission;
        }
    }

    event routeTerminated(string indexed destination);

    function terminateRoute(
        string memory destination
    ) public {
        if (canAuthorize(destination)) {
            routes[destination].exists = false;
        }
    }

    // Exec Query
    event FlightPlan(string indexed destination, string query, address operator);

    function fileFlightPlan(string memory destination, string memory query) public {
        if (routes[destination].exists) {
            emit FlightPlan(destination, query, msg.sender);
        }
    }
}
