var mamidApp = angular.module('mamidApp', ['ngRoute', 'ngResource', 'ngSanitize']);


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
                var root = $('#content')[0];
                var ediv = document.createElement('div');
                ediv.setAttribute('class', 'alert alert-danger alert-dismissible fade in');
                ediv.setAttribute('role', 'alert');
                var button = document.createElement('button');
                button.setAttribute('class', 'close');
                button.setAttribute('data-dismiss', 'alert');
                button.setAttribute('type', 'button');
                button.innerHTML = '<span aria-hidden="true">&times;</span>';
                ediv.appendChild(button);
                var h4 = document.createElement('h4');
                h4.innerHTML = '<span class="glyphicon glyphicon-warning-sign"></span> An error occurred.';
                ediv.appendChild(h4);
                var p = document.createElement('p');
                if (rejection.data != null)
                    p.innerHTML = rejection.data;
                else
                    p.innerHTML = "A fatal application error occurred (maybe connection to the server lost?). You may reload the page.";
                ediv.appendChild(p);
                $(ediv).hide();
                root.insertBefore(ediv, root.firstChild);
                $(ediv).alert();
                $(ediv).fadeTo(5000, 500).slideUp(500, function () {
                    $(ediv).alert('close');
                });
                window.console.error(rejection);
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
        .when('/system', {
            templateUrl: 'pages/system.html',
            controller: 'systemController'
        })
        .when('/about', {
            templateUrl: 'pages/about.html'
        })
});

mamidApp.factory('SlaveService', function ($resource) {
    return $resource('/api/slaves/:slave', {slave: "@id"}, {
        create: {method: 'put'},
        queryByReplicaSet: {method: 'get', url: '/api/replicasets/:replicaset/slaves/', isArray: true},
        getMongods: {method: 'get', url: '/api/slaves/:slave/mongods', isArray: true}
    });
});

mamidApp.factory('ProblemService', function ($resource) {
    return $resource('/api/problems/:problem', {problem: "@id"}, {});
});

mamidApp.factory('KeyFileService', function ($resource) {
    return $resource('/api/system/keyfile', {}, {});
});

mamidApp.factory('MgmtUserService', function ($resource) {
    return $resource('/api/system/managementuser', {}, {});
});

mamidApp.factory('ReplicaSetService', function ($resource) {
    return $resource('/api/replicasets/:replicaset', {replicaset: "@id"}, {
        create: {method: 'put'},
        getMongods: {method: 'get', url: '/api/replicasets/:replicaset/mongods', isArray: true}
    });
});

mamidApp.factory('RiskGroupService', function ($resource) {
    return $resource('/api/riskgroups/:riskgroup', {riskgroup: "@id"}, {
        create: {method: 'put'},
        getUnassignedSlaves: {method: 'get', url: '/api/riskgroups/null/slaves/', isArray: true},
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

mamidApp.controller('mainController', function ($scope, $location, $timeout, filterFilter, SlaveService, ProblemService) {
    $scope.problemsBySlave = {};
    $scope.problemsByReplicaSet = {};
    chart = null;
    $scope.getStateCount = function (state) {
        var s = [];
        for (var i = 0; i < $scope.slaves.length; i++) {
            if (!$scope.problemsBySlave[$scope.slaves[i].id + ""]) {
                s.push($scope.slaves[i]);
            }
        }
        return filterFilter(s, {configured_state: state}).length;
    };

    $scope.getProblemCount = function () {
        var count = 0;
        for (var i = 0; i < $scope.slaves.length; i++) {
            if ($scope.slaves[i].id + "" in $scope.problemsBySlave && $scope.problemsBySlave[$scope.slaves[i].id + ""].length > 0) {
                count++;
            }
        }
        return count;
    };
    var genChart = function () {
        if (chart != null) {
            var probs = $scope.getProblemCount();
            (function (probs) {
                chart.load({
                    columns: [
                        ['Active', $scope.getStateCount('active')],
                        ['Maintenance', $scope.getStateCount('maintenance')],
                        ['Disabled', $scope.getStateCount('disabled')],
                        ['Problematic', probs]
                    ]
                })
            })(probs);
            return;
        }
        chart = c3.generate({
            bindto: '#slaves',
            data: {
                columns: [
                    ['Active', $scope.getStateCount('active')],
                    ['Maintenance', $scope.getStateCount('maintenance')],
                    ['Disabled', $scope.getStateCount('disabled')],
                    ['Problematic', $scope.getProblemCount()]
                ],
                type: 'donut',
            },
            donut: {
                title: "Slave states"
            }
            ,
            color: {
                pattern: ['#22aa22', '#0af', '#f0ad4e', '#d43f3a']
            }
            ,
        })
        ;
    };
    $scope.slaves = SlaveService.query();
    $scope.codeReplace = function (string) {
        return string.replace(
            /`(\w*)`/gi,
            '<code>$1</code>'
        );
    };
    if (!problemPolling) {
        (function tick() {
            problemPolling = true;
            ProblemService.query(function (problems) {
                    $scope.problems = problems;
                    $scope.problemsBySlave = {};
                    $scope.problemsByReplicaSet = {};
                    for (var i = 0; i < $scope.problems.length; i++) {
                        $scope.problems[i].description = $scope.codeReplace($scope.problems[i].description);
                        $scope.problems[i].long_description = $scope.codeReplace($scope.problems[i].long_description);
                        if ($scope.problems[i].replica_set_id != null) {
                            if (!($scope.problems[i].replica_set_id + "" in $scope.problemsByReplicaSet)) {
                                $scope.problemsByReplicaSet[$scope.problems[i].replica_set_id + ""] = [];
                            }
                            $scope.problemsByReplicaSet[$scope.problems[i].replica_set_id + ""].push($scope.problems[i]);
                        }
                        if ($scope.problems[i].slave_id != null) {
                            if (!($scope.problems[i].slave_id in $scope.problemsBySlave)) {
                                $scope.problemsBySlave[$scope.problems[i].slave_id + ""] = [];
                            }
                            $scope.problemsBySlave[$scope.problems[i].slave_id + ""].push($scope.problems[i]);
                        }
                    }
                    $timeout(tick, 1000 * 5);
                }
            );
        })();
    }
    $scope.$location = $location;
    $scope.$watch('problemsBySlave', function () {
        genChart();
    });

});

mamidApp.controller('slaveIndexController', function ($scope, $http, $timeout, SlaveService) {
    $scope.loading = true;
    SlaveService.query(function (slaves) {
        $scope.slaves = slaves;
        $scope.loading = false;
        $(function () {
            $.each($('.mamid-collapse'), function (index, value) {
                value = $(value);
                value.click(function () {
                    $.each(value.find('.caret'), function (index, value) {
                        value = $(value);
                        if (value.hasClass('rotatecaret')) {
                            value.addClass('rotatecaret-up');
                            value.removeClass('rotatecaret');
                        } else {
                            value.addClass('rotatecaret');
                            value.removeClass('rotatecaret-up');
                        }
                    });
                });

            });
        });
    });

});
var problemPolling = false;
mamidApp.controller('problemIndexController', function ($scope, $http, $timeout, ProblemService) {
    $scope.formatDate = function (date) {
        return String(new Date(Date.parse(date)));
    }
});

mamidApp.controller('riskGroupIndexController', function ($scope, $http, RiskGroupService) {
    $scope.riskgroups = RiskGroupService.query();
    $scope.unassigned_slaves = RiskGroupService.getUnassignedSlaves();
    $scope.new_riskgroup = new RiskGroupService();
    $scope.createRiskGroup = function () {
        $scope.new_riskgroup.$create(function () {
            $scope.new_riskgroup = new RiskGroupService();
        });
    };
    $scope.assignToRiskGroup = function (slave, oldriskgroup) {
        if (slave.riskgroup == 0) {
            RiskGroupService.removeFromRiskGroup({slave: slave.id, riskgroup: oldriskgroup.id});
            $scope.refreshRiskGroups();
            return;
        }
        RiskGroupService.assignToRiskGroup({slave: slave.id, riskgroup: slave.riskgroup});
        $scope.refreshRiskGroups();
    };
    $scope.getSlaves = function (riskgroup) {
        RiskGroupService.getSlaves({riskgroup: riskgroup.id}, function (slaves) {
            riskgroup.slaves = slaves;
        });
    };
    $scope.removeRiskGroup = function (riskgroup) {
        RiskGroupService.remove({riskgroup: riskgroup.id});
        $scope.refreshRiskGroups();
        $('#confirm_remove' + riskgroup.id).modal('hide');
    };
    $scope.isDeletable = function (riskgroup) {
        if (riskgroup.slaves === undefined) {
            $scope.getSlaves(riskgroup);
        }
        return riskgroup.slaves.length == 0;
    };
    $(function () {
        $('[data-toggle="tooltip"]').tooltip();
    });

    $scope.refreshRiskGroups = function () {
        RiskGroupService.query(function (riskgroups) {
            RiskGroupService.getUnassignedSlaves(function (unassigned) {
                $scope.riskgroups = riskgroups;
                $scope.unassigned_slaves = unassigned;
            });
        });
    }
});
var slaveMongoPoll = false;
mamidApp.controller('slaveByIdController', function ($scope, $http, $routeParams, $location, $timeout, SlaveService, RiskGroupService, ReplicaSetService) {
    var slaveId = $routeParams['slaveId'];
    slaveMongoPoll = !$scope.is_create_view;
    var pollMongo = function () {
        if (!slaveMongoPoll) {
            return;
        }
        SlaveService.getMongods({slave: $scope.slave.id}, function (mongods) {
            SlaveService.get({slave: $scope.slave.id}, function (slave) {
                $scope.slave.$promise.then(function () {
                    $scope.slave.configured_state_transitioning = slave.configured_state_transitioning;
                });
            });
            var finished = 0;
            if (mongods.length == 0) {
                $scope.mongods = mongods;
            }
            for (var i = 0; i < mongods.length; i++) {
                if (mongods[i].replica_set_id != 0) {
                    (function (i) {
                        ReplicaSetService.get({replicaset: mongods[i].replica_set_id}, function (repli) {
                            mongods[i].replicaset = repli;
                            finished++;
                            if (finished == mongods.length) {
                                $scope.mongods = mongods;
                            }
                        });
                    })(i);
                } else {
                    finished++;
                }
            }

            $timeout(pollMongo, 2000);
        });
    };

    $scope.is_create_view = slaveId === "new";
    if ($scope.is_create_view) {
        $scope.slave = new SlaveService();

        $scope.slave.configured_state = "disabled";
        $scope.riskgroups = RiskGroupService.query()
        //Copy slave for edit form so that changes are only applied to model when apply is clicked
        $scope.edit_slave = angular.copy($scope.slave);
        $scope.edit_slave.new_riskgroup_id = "";
    } else {
        $scope.slave = SlaveService.get({slave: slaveId});

        //Copy slave for edit form so that changes are only applied to model when apply is clicked
        $scope.slave.$promise.then(function () {
            if ($scope.slave.risk_group_id != null) {
                $scope.slave.riskgroup = RiskGroupService.get({riskgroup: $scope.slave.risk_group_id});
            }
            pollMongo();
            $scope.edit_slave = angular.copy($scope.slave);
        });
    }

    $scope.$on('$destroy', function () {
        slaveMongoPoll = false;
    });

    $scope.updateSlave = function () {
        if ($scope.is_create_view) {
            if (!$scope.edit_slave.slave_port) {
                $scope.edit_slave.slave_port = 8081;
            }
            if (!$scope.edit_slave.mongod_port_range_begin) {
                $scope.edit_slave.mongod_port_range_begin = 18080;
            }
            if (!$scope.edit_slave.mongod_port_range_end) {
                $scope.edit_slave.mongod_port_range_end = 18081;
            }
            angular.copy($scope.edit_slave, $scope.slave);
            $scope.slave.$create(function (res) {
                if ($scope.edit_slave.new_riskgroup_id != "") {
                    RiskGroupService.assignToRiskGroup({slave: res.id, riskgroup: $scope.edit_slave.new_riskgroup_id}, function () {
                        $location.path("/slaves/" + res.id);
                    });
                } else {
                    $location.path("/slaves/" + res.id);
                }
            });
        } else {
            $scope.edit_slave.$save(function (slave) {
                $scope.slave = slave;
                $scope.slave.$promise.then(function () {
                    $scope.edit_slave = angular.copy($scope.slave);
                });
            });
        }

    };

    $scope.deleteSlave = function () {
        $scope.slave.$delete(function () {
            $location.path("/slaves");
        });
        $('#confirm_remove').modal('hide');
    };

    $scope.setSlaveState = function (state) {
        $scope.slave.configured_state = state;
        $scope.edit_slave.configured_state = state;
        $scope.slave.$save(function () {
            SlaveService.get({slave: slaveId}, function (slave) {
                $scope.slave = slave;
                $scope.slave.$promise.then(function () {
                    $scope.edit_slave = angular.copy($scope.slave);
                    $('#confirm_disable').modal('hide');
                });
            });
        });
    };

    $scope.calcMongodCount = function () {
        if (!$scope.edit_slave) {
            return '?';
        }
        var begin = $scope.edit_slave.mongod_port_range_begin;
        var end = $scope.edit_slave.mongod_port_range_end;
        if (begin + "" === 'undefined' || begin === null)
            begin = 18080;
        if (end + "" === 'undefined' || end == null)
            end = 18081;
        return end - begin;
    };

    $(function () {
        $('[data-toggle="tooltip"]').tooltip();
    });

});

mamidApp.controller('replicasetIndexController', function ($scope, $http, ReplicaSetService) {
    $scope.loading = true;
    ReplicaSetService.query(function (replicasets) {
        $scope.replicasets = replicasets;
        $scope.loading = false;
    });
});

var replicaSetMongoPoll = false;

mamidApp.controller('replicasetByIdController',
    function ($scope, $http, $routeParams, $location, $timeout, SlaveService, ReplicaSetService) {
        var replicasetId = $routeParams['replicasetId'];
        replicaSetMongoPoll = !$scope.is_create_view;
        var mongoPoll = function () {
            if (!replicaSetMongoPoll) {
                return;
            }
            ReplicaSetService.getMongods({replicaset: replicasetId}, function (mongods) {
                SlaveService.queryByReplicaSet({replicaset: replicasetId}, function (slaves) {
                    for (var i = 0; i < mongods.length; i++) {
                        for (var j = 0; j < slaves.length; j++) { // we want a dict here, but... meh...
                            if (slaves[j].id == mongods[i].parent_slave_id) {
                                mongods[i].slave = slaves[j]
                            }
                        }
                    }
                    $scope.replicaset.mongods = mongods;
                    $timeout(mongoPoll, 2000);
                });


            });
        };
        $scope.is_create_view = replicasetId === "new";
        if ($scope.is_create_view) {
            $scope.replicaset = new ReplicaSetService();

            //Copy replicaset for edit form so that changes are only applied to model when apply is clicked
            $scope.edit_replicaset = angular.copy($scope.replicaset);
            $scope.edit_replicaset.sharding_role = "none";
        } else {
            $scope.replicaset = ReplicaSetService.get({replicaset: replicasetId});

            $scope.replicaset.$promise.then(function () {
                $scope.edit_replicaset = angular.copy($scope.replicaset);
                mongoPoll();

            });

        }
        $scope.$on("$destroy", function () {
            replicaSetMongoPoll = false;
        });
        $scope.updateReplicaSet = function () {
            angular.copy($scope.edit_replicaset, $scope.replicaset);
            if ($scope.is_create_view) {
                $scope.replicaset.$create(function (res) {
                    $location.path("/replicasets/" + res.id);
                });
            } else {
                $scope.replicaset.$save();
            }
        };

        $scope.deleteReplicaSet = function () {
            $scope.replicaset.$delete(function () {
                $location.path("/replicasets");
            });
            $('#confirm_remove').modal('hide');
        };
        $scope.setShardingRole = function (role) {
            if ($scope.is_create_view) {
                $scope.edit_replicaset.sharding_role = role;
            }
        }
        $scope.generateMongoCLIString = function () {
            if (!('mongods' in $scope.replicaset))
                return "[mongo1:port,mongo2:port,...]";
            var res = "";
            for (var i = 0; i < $scope.replicaset.mongods.length; i++) {
                res += $scope.replicaset.mongods[i].slave.hostname + ':' + $scope.replicaset.mongods[i].slave_port + ","
            }
            res = res.substr(0, res.length - 1);
            return res;
        }
    });

mamidApp.controller('systemController', function ($scope, $http, KeyFileService, MgmtUserService) {
    var c = 0;
    $scope.e = function () {
        if (c == 5) {
            var l = $('.mamid-logo');
            var co = $('.content');
            co.attr("style", "overflow-x: hidden;");
            l.animate({
                "padding-left": "+=80vw",
            }, 500, "swing", function () {
                l.attr("style", "display: none;");
                co.attr("style", "");
            });
        }
        c++;
    }
    $scope.keyFile = KeyFileService.get();
    $scope.mgmtUser = MgmtUserService.get();

});
