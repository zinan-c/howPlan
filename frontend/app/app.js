(function () {
  'use strict';

  angular.module('travelPlannerApp', ['ngRoute']).config(['$routeProvider', function ($routeProvider) {
    $routeProvider
      .when('/plans', {
        templateUrl: 'app/views/plans.html',
        controller: 'PlansController',
        controllerAs: 'vm'
      })
      .when('/plan/:id', {
        templateUrl: 'app/views/map.html',
        controller: 'MapController',
        controllerAs: 'vm'
      })
      .when('/map', { redirectTo: '/plans' })
      .when('/', { redirectTo: '/plans' })
      .otherwise({ redirectTo: '/plans' });
  }]);
})();
