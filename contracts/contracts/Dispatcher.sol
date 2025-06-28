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

    bytes16[] public publicAirportCodes;
 
    function viewPublicAirportCodes() public view returns (bytes16[] memory) {
        return airportCodes;
    }

    struct Airport {
        address owner;
        bytes4 ipAddr;
        uint index1;
        uint index2;
        bool available;
        bool exists;
    }
    mapping(bytes16 => Airport) public airports;

    function viewAirport(bytes16 airportCode) public view returns (
        address owner,
        bytes4 ipAddr,
        uint index1,
        uint index2,
        bool available,
        bool exists
    ) {
        Airport memory airport = airports[airportCode];
        return (airport.owner, airport.ipAddr, airport.index1, airport.index2, airport.available, airport.exists);
    }

    function constructAirport(
        bytes16 airportCode,
        bytes4 ipAddr
    ) public {
        if (!airports[airportCode].exists) {
            airportCodes.push(airportCode);

            uint index2;
            if (ipAddr != 0) {
                publicAirportCodes.push(airportCode);
                index2 = publicAirportCodes.length - 1;
            }

            airports[airportCode] = Airport({
                owner: msg.sender,
                ipAddr: ipAddr,
                index1: airportCodes.length - 1,
                index2: index2,
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
        airports[airportCode].index1 = airportCodes.length - 1;
        if (airports[airportCode].ipAddr != 0) {
            publicAirportCodes.push(airportCode);
            airports[airportCode].index2 = publicAirportCodes.length - 1;
        }

        airports[airportCode].available = true;
    }

    function closeAirport(
        bytes16 airportCode
    ) public {
        require(airports[airportCode].owner == msg.sender, "You are not owner.");

        uint index = airports[airportCode].index1;
        uint lastIndex = airportCodes.length - 1;
        (airportCodes[index], airportCodes[lastIndex]) = (airportCodes[lastIndex], airportCodes[index]);
        
        airports[airportCode].index1 = lastIndex;
        airports[airportCodes[index]].index1 = index;
        
        airportCodes.pop();
        airports[airportCode].available = false;

        if (airports[airportCode].ipAddr != 0) {
            index = airports[airportCode].index2;
            lastIndex = publicAirportCodes.length - 1;
            (publicAirportCodes[index], publicAirportCodes[lastIndex]) = (publicAirportCodes[lastIndex], publicAirportCodes[index]);
        
            airports[airportCode].index2 = lastIndex;
            airports[publicAirportCodes[index]].index2 = index;
        
            publicAirportCodes.pop();
        }
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
        uint nonce;
        address[] canAuthorizeAccounts;
        address[] canWriteAccounts;
        string defaultPermission;
        bool exists;
    }
    mapping(bytes16 => Route) public routes;

    event NewRouteLaunched(bytes16 indexed origin, bytes16 destination, address[] canWriteAccount, string defaultPermission, bool reroute, bytes4 establishedOriginIPAddr);
    
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
            nonce: 0,
            canAuthorizeAccounts: canAuthorizeAccounts,
            canWriteAccounts: canWriteAccounts,
            defaultPermission: defaultPermission,
            exists: true
        });

        for (uint i = 0; i < 5; i++) {
            emit NewRouteLaunched(origins[i], destination, canWriteAccounts, defaultPermission, false, 0);
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

    event PermissionChanged(bytes16 indexed destination, address[] canWriteAccount, string defaultPermission);

    function changePermission(
        bytes16 destination,
        address[] memory canWriteAccounts,
        string memory defaultPermission
    ) public {
        require(canAuthorize(destination), "You can not authorize.");

        routes[destination].canWriteAccounts = canWriteAccounts;
        routes[destination].defaultPermission = defaultPermission;

        emit PermissionChanged(destination, canWriteAccounts, defaultPermission);
    }

    event RouteTerminated(bytes16 indexed origin, bytes16 destination);

    function terminateRoute(
        bytes16 destination
    ) public {
        require(canAuthorize(destination), "You can not authorize.");
        for (uint i = 0; i < 5; i++) {
            emit RouteTerminated(routes[destination].origins[i], destination);
        }
    }

    event Rerouted(bytes16 indexed establishedOrigin, bytes4 newOriginIPAddr);

    function reroute(bytes16 destination, bytes16 distablishedOrigin) internal {
        require(airportCodes.length >= 6, "Need at least 6 available airports.");
        // originsからdistablishedOriginを削除
        bytes16 establishedOrigin;
        uint distablishedOriginIndex;

        for (uint i = 0; i < 5; i++) {
            if (routes[destination].origins[i] == distablishedOrigin) {
                distablishedOriginIndex = i;
            } else if (airports[routes[destination].origins[i]].ipAddr != 0) {
                establishedOrigin = routes[destination].origins[i];
            }
        }
        
        // routeTerminatedをdistablished Originに送信
        emit RouteTerminated(distablishedOrigin, destination);

        // OriginsのいずれかにIPが登録されていれば、airportsからnewOriginをランダムに選出しNewRouteLaunchedイベントを発行
        // されていなければpublicAiportsから選出しRerouteイベントを発行
        bytes16 newOrigin;
        if (establishedOrigin == 0) {
            // originsの全てのairportにipが全て設定されていない場合なので被らない。
            newOrigin = publicAirportCodes[pseudoRandom() % publicAirportCodes.length];
        } else {
            bytes16[] memory pool = airportCodes;

            for (uint i = 0; i < 5; i++) {
                uint j = airports[routes[destination].origins[i]].index1;
                uint k = airportCodes.length - 1 - i;
                (pool[j], pool[k]) = (pool[k], pool[j]);
            }
            newOrigin = pool[pseudoRandom() % pool.length - 5];
        }
        
        if (establishedOrigin == 0) {
            emit Rerouted(
                routes[destination].origins[0],
                airports[newOrigin].ipAddr
            );
        } else {
            emit NewRouteLaunched(
                newOrigin,
                destination,
                routes[destination].canWriteAccounts,
                routes[destination].defaultPermission,
                true,
                airports[establishedOrigin].ipAddr
            );
        }
    }

    // Exec Query
    event FlightPlanSubmitted(bytes16 indexed destination, uint nonce, string query, address operator);

    function submitFlightPlan(bytes16 destination, string memory query) public {
        require(routes[destination].exists, "Route does not exist.");
        routes[destination].nonce = routes[destination].nonce + 1;
        emit FlightPlanSubmitted(destination, routes[destination].nonce + 1, query, msg.sender);
    }
}
