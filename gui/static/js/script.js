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
        .when('/replicasets', {
            templateUrl: 'pages/replicasets.html',
            controller: 'replicasetIndexController'
        })
        .when('/replicasets/:replicasetId', {
            templateUrl: 'pages/replicaset.html',
            controller: 'replicasetByIdController'
        })
        .when('/problems', {
            templateUrl: 'pages/problems.html',
            controller: 'problemIndexController'
        })
        .when('/riskgroups', {
            templateUrl: 'pages/riskgroups.html',
            controller: 'riskGroupIndexController'
        })
});

mamidApp.factory('SlaveService', function ($resource) {
    return $resource('/api/slaves/:slave', {slave: "@id"}, {
        create: {method: 'put'},
        queryByReplicaSet: {method: 'get', url: '/api/replicasets/:replicaset/slaves/'}
    });
});

mamidApp.factory('ProblemService', function ($resource) {
    return $resource('/api/problems/:problem', {problem: "@id"}, {});
});

mamidApp.factory('ReplicaSetService', function ($resource) {
    return $resource('/api/replicasets/:replicaset', {replicaset: "@id"}, {
        create: {method: 'put'}
    });
});

mamidApp.factory('RiskGroupService', function ($resource) {
    return $resource('/api/riskgroups/:riskgroup', {riskgroup: "@id"}, {
        create: {method: 'put'},
        getUnassignedSlaves: {method: 'get', url: '/api/riskgroups/0/slaves/', isArray: true},
        assignToRiskGroup: {
            method: 'put',
            url: '/api/riskgroups/:riskgroup/slaves/:slave',
            params: {riskgroup: "@riskgroup", slave: "@slave"}
        },
        getSlaves: {method: 'get', url: '/api/riskgroups/:riskgroup/slaves', isArray: true},
        remove: {method: 'delete'}
    });
});

mamidApp.controller('mainController', function ($scope) {
    $scope.message = 'Greetings from the controller';
});

mamidApp.controller('slaveIndexController', function ($scope, $http, SlaveService) {
    $scope.slaves = SlaveService.query()
});

mamidApp.controller('problemIndexController', function ($scope, $http, SlaveService) {
    $scope.problems = ProblemService.query()
});

mamidApp.controller('riskGroupIndexController', function ($scope, $http, RiskGroupService) {
    $scope.riskgroups = RiskGroupService.query();
    $scope.unassigned_slaves = RiskGroupService.getUnassignedSlaves();
    $scope.new_riskgroup = new RiskGroupService();
    $scope.createRiskGroup = function () {
        $scope.new_riskgroup.$create();
        $scope.new_riskgroup = null;
        $scope.riskgroups = RiskGroupService.query();
    };
    $scope.assignToRiskGroup = function (slave) {
        RiskGroupService.assignToRiskGroup({slave: slave.id, riskgroup: slave.riskgroup});
        $scope.unassigned_slaves = RiskGroupService.getUnassignedSlaves();
    };
    $scope.getSlaves = function (riskgroup) {
        riskgroup.slaves = RiskGroupService.getSlaves({riskgroup: riskgroup.id});
    }
    $scope.removeRiskGroup = function (riskgroup) {
        riskgroup.slaves = RiskGroupService.remove({riskgroup: riskgroup.id});
        $scope.riskgroups = RiskGroupService.query();
        $('#confirm_remove'+riskgroup.id).modal('hide');
    }
    $(function () {
        $('[data-toggle="tooltip"]').tooltip();
    })
});

mamidApp.controller('slaveByIdController', function ($scope, $http, $routeParams, $location, SlaveService) {
    var slaveId = $routeParams['slaveId'];
    $scope.is_create_view = slaveId === "new";
    if ($scope.is_create_view) {
        $scope.slave = new SlaveService();

        $scope.slave.configured_state = "disabled";

        //Copy slave for edit form so that changes are only applied to model when apply is clicked
        $scope.edit_slave = angular.copy($scope.slave);
    } else {
        $scope.slave = SlaveService.get({slave: slaveId});

        //Copy slave for edit form so that changes are only applied to model when apply is clicked
        $scope.slave.$promise.then(function () {
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
        $scope.slave.configured_state = state;
        $scope.slave.$save();
    }
});

mamidApp.controller('replicasetIndexController', function ($scope, $http, ReplicaSetService) {
    $scope.replicasets = ReplicaSetService.query()
});

mamidApp.controller('replicasetByIdController',
    function ($scope, $http, $routeParams, $location, SlaveService, ReplicaSetService) {
        var replicasetId = $routeParams['replicasetId'];
        $scope.is_create_view = replicasetId === "new";
        if ($scope.is_create_view) {
            $scope.replicaset = new ReplicaSetService();

            //Copy replicaset for edit form so that changes are only applied to model when apply is clicked
            $scope.edit_replicaset = angular.copy($scope.replicaset);
        } else {
            $scope.replicaset = ReplicaSetService.get({replicaset: replicasetId});
            $scope.replicaset_slaves = SlaveService.queryByReplicaSet({replicaset: replicasetId});

            //Copy replicaset for edit form so that changes are only applied to model when apply is clicked
            $scope.replicaset.$promise.then(function () {
                $scope.edit_replicaset = angular.copy($scope.replicaset);
            });
        }

        $scope.updateReplicaSet = function () {
            angular.copy($scope.edit_replicaset, $scope.replicaset);
            if ($scope.is_create_view) {
                $scope.replicaset.$create();
            } else {
                $scope.replicaset.$save();
            }
            $location.path("/replicasets");
        };

        $scope.deleteReplicaSet = function () {
            $scope.replicaset.$delete();
            $('#confirm_remove').modal('hide');
            $location.path("/replicasets");
        };
    });