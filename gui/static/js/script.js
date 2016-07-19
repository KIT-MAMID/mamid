var mamidApp = angular.module('mamidApp', ['ngRoute', 'ngResource']);

mamidApp.config(function ($routeProvider) {
   $routeProvider
       .when('/', {
           templateUrl: 'pages/home.html',
           controller: 'mainController'
       })
       .when('/slaves', {
           templateUrl: 'pages/slaves.html',
           controller: 'slaveIndexController'
       })
       .when('/slaves/:slaveId', {
           templateUrl: 'pages/slave.html',
           controller: 'slaveByIdController'
       })
});

mamidApp.factory('SlaveService', function ($resource) {
    return $resource('/api/slaves/:slave', {slave: "@id"}, {
        create: {method: 'put'}
    });
});

mamidApp.controller('mainController', function($scope) {
    $scope.message = 'Greetings from the controller';
});

mamidApp.controller('slaveIndexController', function($scope, $http, SlaveService) {
    $scope.slaves = SlaveService.query()
});

mamidApp.controller('slaveByIdController', function($scope, $http, $routeParams, $location, SlaveService) {
    var slaveId = $routeParams['slaveId'];
    $scope.is_create_view = slaveId === "new";
    if ($scope.is_create_view) {
        $scope.slave = new SlaveService();

        $scope.slave.state = "disabled";

        //Copy slave for edit form so that changes are only applied to model when apply is clicked
        $scope.edit_slave = angular.copy($scope.slave);
    } else {
        $scope.slave = SlaveService.get({slave: slaveId});
        
        //Copy slave for edit form so that changes are only applied to model when apply is clicked
        $scope.slave.$promise.then(function(){
            $scope.edit_slave = angular.copy($scope.slave);
        });
    }

    $scope.updateSlave = function () {
        angular.copy($scope.edit_slave, $scope.slave);
        if ($scope.is_create_view) {
            $scope.slave.$create();
        } else {
            $scope.slave.$save();
        }
        $location.path("/slaves");
    };

    $scope.deleteSlave = function () {
        $scope.slave.$delete();
        $('#confirm_remove').modal('hide');
        $location.path("/slaves");
    };

    $scope.setSlaveState = function (state) {
        $scope.slave.state = state;
        $scope.slave.$save();
    }
});