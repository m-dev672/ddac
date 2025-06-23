// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

// Uncomment this line to use console.log
// import "hardhat/console.sol";

contract Dispatcher {
    // About Node
    bytes16[] public airportCodes;
 
    function viewAirportCodes() public view returns (bytes16[] memory) {
        return airportCodes;
    }

    struct Airport {
        address owner;
        uint index;
        bool available;
        bool exists;
    }
    mapping(bytes16 => Airport) public airports;

    function viewAirport(bytes16 airportCode) public view returns (
        address owner,
        uint index,
        bool available,
        bool exists
    ) {
        Airport memory airport = airports[airportCode];
        return (airport.owner, airport.index, airport.available, airport.exists);
    }

    function constructAirport(
        bytes16 airportCode
    ) public {
        if (!airports[airportCode].exists) {
            airportCodes.push(airportCode);

            airports[airportCode] = Airport({
                owner: msg.sender,
                index: airportCodes.length - 1,
                available: true,
                exists: true
            });
        }
    }

    function openAirport(
        bytes16 airportCode
    ) public {
        require(airports[airportCode].owner == msg.sender, "You are not owner.");
        require(!airports[airportCode].available, "Your airport is available.");

        airportCodes.push(airportCode);
        airports[airportCode].index = airportCodes.length - 1;

        airports[airportCode].available = true;
        
    }

    function closeAirport(
        bytes16 airportCode
    ) public {
        require(airports[airportCode].owner == msg.sender, "You are not owner.");

        uint index = airports[airportCode].index;
        uint lastIndex = airportCodes.length - 1;
        (airportCodes[index], airportCodes[lastIndex]) = (airportCodes[lastIndex], airportCodes[index]);
        
        airports[airportCode].index = lastIndex;
        airports[airportCodes[index]].index = index;
        
        airportCodes.pop();
        airports[airportCode].available = false;
    }

    function destroyAirport(
        bytes16 airportCode
    ) public {
        require(airports[airportCode].owner == msg.sender, "You are not owner.");

        airports[airportCode].exists = false;
    }

    // About Database
    uint seed;
    function pseudoRandom() internal returns (uint) {
        seed++;
        return uint(keccak256(abi.encodePacked(block.prevrandao, block.timestamp, msg.sender, seed)));
    }

    struct Route {
        bytes16[5] origins;
        address[] canAuthorizeAccounts;
        address[] canWriteAccounts;
        string defaultPermission;
        bool exists;
    }
    mapping(bytes16 => Route) public routes;

    event NewRouteLaunched(bytes16 indexed origin, bytes16 destination, address[] canWriteAccount, string defaultPermission);
    
    function launchNewRoute(
        bytes16 destination,
        address[] memory canAuthorizeAccounts,
        address[] memory canWriteAccounts,
        string memory defaultPermission
    ) public {
        require(!routes[destination].exists, "Route already exist.");
        require(airportCodes.length >= 5, "Need at least 5 available airports.");

        uint n = airportCodes.length;
        bytes16[] memory pool = airportCodes;
        bytes16[5] memory origins;

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
            emit NewRouteLaunched(origins[i], destination, canWriteAccounts, defaultPermission);
        }
    }

    function canAuthorize(bytes16 destination) internal view returns (bool) {
        address[] memory canAuthorizeAccounts = routes[destination].canAuthorizeAccounts;

        for (uint i = 0; i < canAuthorizeAccounts.length; i++) {
            if (canAuthorizeAccounts[i] == msg.sender) {
                return true;
            }
        }

        return false;
    }

    event permissionChanged(bytes16 indexed destination, address[] canWriteAccount, string defaultPermission);

    function changePermission(
        bytes16 destination,
        address[] memory canWriteAccounts,
        string memory defaultPermission
    ) public {
        require(canAuthorize(destination), "You can not authorize.");

        routes[destination].canWriteAccounts = canWriteAccounts;
        routes[destination].defaultPermission = defaultPermission;

        emit permissionChanged(destination, canWriteAccounts, defaultPermission);
    }

    event routeTerminated(bytes16 indexed origin, bytes16 destination);

    function terminateRoute(
        bytes16 destination
    ) public {
        require(canAuthorize(destination), "You can not authorize.");
        for (uint i = 0; i < 5; i++) {
            emit routeTerminated(routes[destination].origins[i], destination);
        }
    }

    // Exec Query
    event FlightPlanSubmitted(bytes16 indexed destination, string query, address operator);

    function submitFlightPlan(bytes16 destination, string memory query) public {
        require(routes[destination].exists, "Route does not exist.");
        emit FlightPlanSubmitted(destination, query, msg.sender);
    }
}
