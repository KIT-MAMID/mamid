var mamidApp = angular.module('mamidApp', ['ngRoute', 'ngResource']);


// http://www.codelord.net/2014/06/25/generic-error-handling-in-angularjs/
var specificallyHandleInProgress = false;
var HEADER_NAME = 'MyApp-Handle-Errors-Generically';
mamidApp.factory('RequestsErrorHandler', ['$q', function ($q) {
    return {
        // --- The user's API for claiming responsiblity for requests ---
        specificallyHandled: function (specificallyHandledBlock) {
            specificallyHandleInProgress = true;
            try {
                return specificallyHandledBlock();
            } finally {
                specificallyHandleInProgress = false;
            }
        },

        // --- Response interceptor for handling errors generically ---
        responseError: function (rejection) {
            var shouldHandle = (rejection && rejection.config && rejection.config.headers
            && rejection.config.headers[HEADER_NAME]);
            if (shouldHandle) {
                window.console.error(rejection);
                window.alert(rejection.data);
            }

            return $q.reject(rejection);
        }
    };
}]);
mamidApp.config(['$provide', '$httpProvider', function ($provide, $httpProvider) {
    $httpProvider.interceptors.push('RequestsErrorHandler');

    // --- Decorate $http to add a special header by default ---

    function addHeaderToConfig(config) {
        config = config || {};
        config.headers = config.headers || {};

        // Add the header unless user asked to handle errors himself
        if (!specificallyHandleInProgress) {
            config.headers[HEADER_NAME] = true;
        }

        return config;
    }

    // The rest here is mostly boilerplate needed to decorate $http safely
    $provide.decorator('$http', ['$delegate', function ($delegate) {
        function decorateRegularCall(method) {
            return function (url, config) {
                return $delegate[method](url, addHeaderToConfig(config));
            };
        }

        function decorateDataCall(method) {
            return function (url, data, config) {
                return $delegate[method](url, data, addHeaderToConfig(config));
            };
        }

        function copyNotOverriddenAttributes(newHttp) {
            for (var attr in $delegate) {
                if (!newHttp.hasOwnProperty(attr)) {
                    if (typeof($delegate[attr]) === 'function') {
                        newHttp[attr] = function () {
                            return $delegate[attr].apply($delegate, arguments);
                        };
                    } else {
                        newHttp[attr] = $delegate[attr];
                    }
                }
            }
        }

        var newHttp = function (config) {
            return $delegate(addHeaderToConfig(config));
        };

        newHttp.get = decorateRegularCall('get');
        newHttp.delete = decorateRegularCall('delete');
        newHttp.head = decorateRegularCall('head');
        newHttp.jsonp = decorateRegularCall('jsonp');
        newHttp.post = decorateDataCall('post');
        newHttp.put = decorateDataCall('put');

        copyNotOverriddenAttributes(newHttp);

        return newHttp;
    }]);
}]);



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
        removeFromRiskGroup: {
            method: 'delete',
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

mamidApp.controller('problemIndexController', function ($scope, $http, $timeout, ProblemService) {
    (function tick() {
        ProblemService.query(function (problems) {
            $scope.problems = problems;
            $timeout(tick, 1000 * 5);
        });
    })();
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
    $scope.assignToRiskGroup = function (slave, oldriskgroup) {
        if (slave.riskgroup == 0) {
            RiskGroupService.removeFromRiskGroup({slave: slave.id, riskgroup: oldriskgroup.id});
            $scope.riskgroups = RiskGroupService.query();
            $scope.unassigned_slaves = RiskGroupService.getUnassignedSlaves();
            return;
        }
        RiskGroupService.assignToRiskGroup({slave: slave.id, riskgroup: slave.riskgroup});
        $scope.riskgroups = RiskGroupService.query();
        $scope.unassigned_slaves = RiskGroupService.getUnassignedSlaves();
    };
    $scope.getSlaves = function (riskgroup) {
        riskgroup.slaves = RiskGroupService.getSlaves({riskgroup: riskgroup.id});
    }
    $scope.removeRiskGroup = function (riskgroup) {
        riskgroup.slaves = RiskGroupService.remove({riskgroup: riskgroup.id});
        $scope.riskgroups = RiskGroupService.query();
        $('#confirm_remove' + riskgroup.id).modal('hide');
    }
    $scope.isDeletable = function (riskgroup) {
        if (riskgroup.slaves === undefined) {
            $scope.getSlaves(riskgroup);
        }
        return riskgroup.slaves.length == 0;
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