(function () {
  'use strict';

  angular.module('travelPlannerApp').service('TripService', ['$http', function ($http) {
    var apiBase = '/api';
    var plansBaseURL = apiBase + '/plans';

    this.getAdminStatus = function (adminOverride) {
      return $http.get(apiBase + '/admin/status?admin=' + adminOverride);
    };

    this.getAllPlans = function () {
      return $http.get(plansBaseURL);
    };

    this.createPlan = function (plan, admin) {
      return $http.post(plansBaseURL + '?admin=' + admin, plan);
    };

    this.getPlanById = function (planId) {
      return $http.get(plansBaseURL + '/' + encodeURIComponent(planId));
    };

    this.updatePlan = function (planId, plan, admin) {
      return $http.put(plansBaseURL + '/' + encodeURIComponent(planId) + '?admin=' + admin, plan);
    };

    this.deletePlan = function (planId, admin) {
      return $http.delete(plansBaseURL + '/' + encodeURIComponent(planId) + '?admin=' + admin);
    };

    this.addStop = function (planId, payload, admin) {
      return $http.post(plansBaseURL + '/' + encodeURIComponent(planId) + '/stops?admin=' + admin, payload);
    };

    this.deleteStop = function (planId, stopId, admin) {
      return $http.delete(plansBaseURL + '/' + encodeURIComponent(planId) + '/stops/' + encodeURIComponent(stopId) + '?admin=' + admin);
    };

    this.getImportTemplateURL = function (admin) {
      return plansBaseURL + '/import/template?admin=' + admin;
    };

    this.importPlan = function (file, planName, admin, onProgress) {
      var fd = new FormData();
      fd.append('file', file);
      if (planName) fd.append('planName', planName);

      return $http.post(plansBaseURL + '/import?admin=' + admin, fd, {
        transformRequest: angular.identity,
        headers: { 'Content-Type': undefined },
        uploadEventHandlers: {
          progress: function (evt) {
            if (!evt || !evt.lengthComputable || !onProgress) return;
            var percent = Math.round((evt.loaded / evt.total) * 100);
            onProgress(percent);
          }
        }
      });
    };
  }]);
})();
