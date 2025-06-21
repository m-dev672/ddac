// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

// Uncomment this line to use console.log
// import "hardhat/console.sol";

contract Dispatcher {
    // Register Node
    string[] public airportCodes;
    string[] public availableAirportCodes;

    struct Airport {
        address owner;
        bool exists;
    }
    mapping(string => Airport) public airports;

    function constructAirport(
        string memory airportCode
    ) public {
        if (!airports[airportCode].exists) {
            airportCodes.push(airportCode);
            availableAirportCodes.push(airportCode);

            airports[airportCode] = Airport({
                owner: msg.sender,
                exists: true
            });
        }
    }

    // Register Database
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

    event NewRoute(string indexed origin, string destination, Route route);
    
    function launchNewRoute(
        string memory destination,
        address[] memory canAuthorizeAccounts,
        address[] memory canWriteAccounts,
        string memory defaultPermission
    ) public {
        if (!routes[destination].exists) {
            require(availableAirportCodes.length >= 5, "Need at least 5 available airports");

            uint n = availableAirportCodes.length;
            string[] memory pool = availableAirportCodes;
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
                emit NewRoute(origins[i], destination, routes[destination]);
            }
        }
    }

    // Exec Query
    event FlightPlan(string indexed destination, string query, address operator);

    function fileFlightPlan(string memory destination, string memory query) public {
        emit FlightPlan(destination, query, msg.sender);
    }
}
