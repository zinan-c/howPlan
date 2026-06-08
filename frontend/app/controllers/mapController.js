(function () {
  'use strict';

  angular.module('travelPlannerApp')
    .controller('ShellController', ['$location', '$window', '$rootScope', 'TripService', function ($location, $window, $rootScope, TripService) {
      var vm = this;
      vm.isAdmin = false;
      vm.currentPlanName = '';
      vm.isPath = function (pathPrefix) {
        return $location.path().indexOf(pathPrefix) === 0;
      };
      vm.exitAdmin = function () {
        var url = new URL($window.location.href);
        forceViewerMode($window);
        url.searchParams.delete('admin');
        $window.location.href = url.toString();
      };
      vm.syncRouteState = function () {
        var path = $location.path();
        if (path.indexOf('/plan/') === 0) {
          var id = decodeURIComponent(path.replace('/plan/', ''));
          TripService.getPlanById(id).then(function (res) {
            vm.currentPlanName = res.data.name || id;
          }).catch(function () {
            vm.currentPlanName = id;
          });
        } else {
          vm.currentPlanName = '';
        }
      };
      vm.refreshAdminState = function () {
        if (readAdminFlag($window)) {
          vm.isAdmin = true;
          return;
        }
        if (isViewerForced($window)) {
          vm.isAdmin = false;
          return;
        }
        TripService.getAdminStatus(readAdminFlag($window)).then(function (res) {
          vm.isAdmin = !!res.data.isAdmin;
        }).catch(function () {
          vm.isAdmin = readAdminFlag($window);
        });
      };

      vm.refreshAdminState();
      vm.syncRouteState();
      $rootScope.$on('$routeChangeSuccess', function () {
        vm.refreshAdminState();
        vm.syncRouteState();
      });
    }])
    .controller('MapController', ['TripService', '$window', '$routeParams', '$location', '$timeout', function (TripService, $window, $routeParams, $location, $timeout) {
      var vm = this;
      vm.planId = $routeParams.id;
      vm.trip = null;
      vm.isAdmin = false;
      vm.error = '';
      vm.adminLabel = 'Viewer';
      vm.selectedDayNumber = null;
      vm.selectedStopId = null;
      vm.focusRequest = null;
      vm.previewStop = null;
      vm.addPointMode = false;
      vm.fixMode = false;
      vm.fixTarget = null;
      vm.fixQueue = [];
      vm.addModal = { show: false, dayNumber: 1, latitude: '', longitude: '', name: '', activityDescription: '', imageUrlsText: '' };
      vm.editModal = { show: false, dayNumber: 1, stopId: '', draft: null };
      vm.editDraft = null;
      vm.planModal = { show: false, name: '', startDate: '', endDate: '', coverImage: '' };
      vm.legendCollapsed = false;
      vm.hasUnlocatedStops = false;
      vm.dayColor = dayColor;
      vm.toggleLegend = function () { vm.legendCollapsed = !vm.legendCollapsed; };
      vm.stopCountLabel = function (count) { return count === 1 ? '1 stop' : count + ' stops'; };
      vm.focusAllStops = function () {
        vm.focusRequest = { mode: 'all', tick: Date.now() };
      };
      vm.focusLegendDay = function (dayNumber) {
        var day = findDay(vm.trip, dayNumber);
        vm.selectedDayNumber = dayNumber;
        vm.selectedStopId = day && day.stops && day.stops.length ? day.stops[0].id : null;
        vm.focusRequest = { mode: 'day', dayNumber: dayNumber, tick: Date.now() };
        scrollAgendaToSelection(dayNumber, vm.selectedStopId);
      };

      vm.backToPlans = function () {
        $location.path('/plans');
      };

      vm.openPlanEditModal = function () {
        if (!vm.ensureAdmin()) return;
        if (!vm.trip) return;
        vm.planModal = {
          show: true,
          name: vm.trip.name,
          startDate: parseISODate(vm.trip.startDate),
          endDate: parseISODate(vm.trip.endDate),
          coverImage: vm.trip.coverImage || ''
        };
      };

      vm.closePlanEditModal = function () {
        vm.planModal.show = false;
      };

      vm.submitPlanBasicEdit = function () {
        if (!vm.ensureAdmin()) return;
        var nextTrip = angular.copy(vm.trip);
        nextTrip.name = vm.planModal.name;
        nextTrip.startDate = formatISODate(vm.planModal.startDate) || vm.trip.startDate;
        nextTrip.endDate = formatISODate(vm.planModal.endDate) || vm.trip.endDate;
        nextTrip.coverImage = vm.planModal.coverImage || '';
        TripService.updatePlan(vm.planId, nextTrip, true).then(function () {
          vm.closePlanEditModal();
          vm.loadTrip();
        }).catch(function (err) {
          vm.error = readError(err, 'Update plan basic info failed');
        });
      };

      vm.ensureAdmin = function () {
        if (vm.isAdmin) return true;
        $window.alert('Write actions are disabled because admin mode is not active.');
        return false;
      };

      vm.isUnlocated = function (stop) {
        return Number(stop.latitude) === 0 && Number(stop.longitude) === 0;
      };

      vm.toggleAddPointMode = function () {
        if (!vm.ensureAdmin()) return;
        vm.fixMode = false;
        vm.fixTarget = null;
        vm.fixQueue = [];
        vm.addPointMode = !vm.addPointMode;
      };

      vm.handleMapClickForAdd = function (payload) {
        if (vm.fixMode && vm.fixTarget) {
          vm.applyFixLocation(payload.lat, payload.lng);
          return;
        }
        if (!vm.addPointMode || !vm.isAdmin) return;
        vm.addModal = {
          show: true,
          dayNumber: vm.selectedDayNumber || 1,
          latitude: payload.lat,
          longitude: payload.lng,
          name: '',
          activityDescription: '',
          imageUrlsText: 'https://via.placeholder.com/320x180?text=New+Stop'
        };
        vm.addPointMode = false;
      };

      vm.startFixSingle = function (dayNumber, stop, $event) {
        if ($event) $event.stopPropagation();
        if (!vm.ensureAdmin()) return;
        vm.addPointMode = false;
        vm.fixMode = true;
        vm.fixQueue = [];
        vm.fixTarget = { dayNumber: dayNumber, stopId: stop.id, name: stop.name };
      };

      vm.startFixAll = function () {
        if (!vm.ensureAdmin()) return;
        var queue = [];
        (vm.trip.dayPlans || []).forEach(function (day) {
          (day.stops || []).forEach(function (stop) {
            if (vm.isUnlocated(stop)) {
              queue.push({ dayNumber: day.dayNumber, stopId: stop.id, name: stop.name });
            }
          });
        });
        if (!queue.length) {
          $window.alert('There are no unlocated stops.');
          return;
        }
        vm.addPointMode = false;
        vm.fixMode = true;
        vm.fixQueue = queue;
        vm.fixTarget = queue.shift();
      };

      vm.cancelFixMode = function () {
        vm.fixMode = false;
        vm.fixTarget = null;
        vm.fixQueue = [];
      };

      vm.applyFixLocation = function (lat, lng) {
        if (!vm.fixTarget) return;
        var nextTrip = angular.copy(vm.trip);
        nextTrip.dayPlans.forEach(function (day) {
          if (day.dayNumber !== vm.fixTarget.dayNumber) return;
          day.stops = day.stops.map(function (stop) {
            if (stop.id !== vm.fixTarget.stopId) return stop;
            stop.latitude = Number(lat.toFixed(6));
            stop.longitude = Number(lng.toFixed(6));
            return stop;
          });
        });
        TripService.updatePlan(vm.planId, nextTrip, true).then(function () {
          if (vm.fixQueue.length) {
            vm.fixTarget = vm.fixQueue.shift();
          } else {
            vm.cancelFixMode();
          }
          vm.loadTrip();
        }).catch(function (err) {
          vm.error = readError(err, 'Failed to fix coordinates');
        });
      };

      vm.markAsComplete = function () {
        if (!vm.ensureAdmin()) return;
        var nextTrip = angular.copy(vm.trip);
        nextTrip.isSimple = false;
        TripService.updatePlan(vm.planId, nextTrip, true).then(function () {
          vm.loadTrip();
        }).catch(function (err) {
          vm.error = readError(err, 'Failed to mark the plan complete');
        });
      };

      vm.submitAddFromMap = function () {
        if (!vm.ensureAdmin()) return;
        var payload = {
          dayNumber: Number(vm.addModal.dayNumber),
          stop: {
            name: vm.addModal.name,
            latitude: Number(vm.addModal.latitude),
            longitude: Number(vm.addModal.longitude),
            activityDescription: vm.addModal.activityDescription || '',
            imageUrls: splitImageUrls(vm.addModal.imageUrlsText)
          }
        };
        TripService.addStop(vm.planId, payload, true).then(function () {
          vm.addModal.show = false;
          vm.loadTrip();
        }).catch(function (err) {
          vm.error = readError(err, 'Add stop failed');
        });
      };

      vm.closeAddModal = function () { vm.addModal.show = false; };
      vm.selectDay = function (dayNumber) { vm.selectedDayNumber = dayNumber; vm.focusRequest = { mode: 'day', dayNumber: dayNumber, tick: Date.now() }; };
      vm.selectStop = function (dayNumber, stop) { vm.selectedDayNumber = dayNumber; vm.selectedStopId = stop.id; vm.focusRequest = { mode: 'stop', dayNumber: dayNumber, stopId: stop.id, tick: Date.now() }; };
      vm.openStopDetails = function (stop) { vm.previewStop = stop; };
      vm.closeStopDetails = function () { vm.previewStop = null; };
      vm.isDayOpen = function (dayNumber) { return vm.selectedDayNumber === dayNumber; };
      vm.handleEditFromMap = function (payload) { if (!vm.ensureAdmin()) return; vm.openEditModal(payload.dayNumber, payload.stop); };
      vm.quickEditStop = function (dayNumber, stop, $event) { if ($event) $event.stopPropagation(); vm.openEditModal(dayNumber, stop); };

      vm.openEditModal = function (dayNumber, stop) {
        if (!vm.ensureAdmin()) return;
        vm.editModal = {
          show: true,
          dayNumber: dayNumber,
          stopId: stop.id,
          draft: {
            id: stop.id,
            name: stop.name,
            latitude: Number(stop.latitude),
            longitude: Number(stop.longitude),
            activityDescription: stop.activityDescription || '',
            imageUrlsText: (stop.imageUrls || []).join(', ')
          }
        };
        vm.editDraft = vm.editModal.draft;
      };

      vm.closeEditModal = function () { vm.editModal.show = false; vm.editDraft = null; };
      vm.handleEditMarkerMoved = function (payload) {
        if (!vm.editModal.show || !vm.editModal.draft) return;
        vm.editModal.draft.latitude = Number(payload.lat.toFixed(6));
        vm.editModal.draft.longitude = Number(payload.lng.toFixed(6));
      };

      vm.submitEditStop = function () {
        if (!vm.ensureAdmin()) return;
        var nextTrip = angular.copy(vm.trip);
        var movedStop = {
          id: vm.editModal.stopId,
          name: vm.editModal.draft.name,
          latitude: Number(vm.editModal.draft.latitude),
          longitude: Number(vm.editModal.draft.longitude),
          activityDescription: vm.editModal.draft.activityDescription || '',
          imageUrls: splitImageUrls(vm.editModal.draft.imageUrlsText)
        };
        nextTrip.dayPlans.forEach(function (day) { day.stops = day.stops.filter(function (s) { return s.id !== vm.editModal.stopId; }); });
        var targetDay = nextTrip.dayPlans.find(function (day) { return day.dayNumber === Number(vm.editModal.dayNumber); });
        if (!targetDay) {
          targetDay = { dayNumber: Number(vm.editModal.dayNumber), title: 'Day ' + vm.editModal.dayNumber, stops: [] };
          nextTrip.dayPlans.push(targetDay);
        }
        targetDay.stops.push(movedStop);
        TripService.updatePlan(vm.planId, nextTrip, true).then(function () {
          vm.closeEditModal();
          vm.loadTrip();
        }).catch(function (err) {
          vm.error = readError(err, 'Update stop failed');
        });
      };

      vm.handleDeleteFromMap = function (payload) {
        if (!vm.ensureAdmin()) return;
        if (!$window.confirm('Delete this stop?')) return;
        TripService.deleteStop(vm.planId, payload.stop.id, true).then(function () {
          if (vm.selectedStopId === payload.stop.id) vm.selectedStopId = null;
          vm.loadTrip();
        }).catch(function (err) {
          vm.error = readError(err, 'Delete failed');
        });
      };

      vm.quickDeleteStop = function (dayNumber, stop, $event) { if ($event) $event.stopPropagation(); vm.handleDeleteFromMap({ dayNumber: dayNumber, stop: stop }); };

      vm.loadAdminStatus = function () {
        if (readAdminFlag($window)) {
          vm.isAdmin = true;
          vm.adminLabel = 'Edit Mode (URL)';
          return;
        }
        if (isViewerForced($window)) {
          vm.isAdmin = false;
          vm.adminLabel = 'Viewer Mode';
          return;
        }
        TripService.getAdminStatus(readAdminFlag($window)).then(function (res) {
          vm.isAdmin = !!res.data.isAdmin;
          vm.adminLabel = vm.isAdmin ? 'Edit Mode' : 'Viewer Mode';
        }).catch(function () {
          vm.isAdmin = readAdminFlag($window);
          vm.adminLabel = vm.isAdmin ? 'Edit Mode (URL)' : 'Viewer Mode';
        });
      };

      vm.loadTrip = function () {
        vm.error = '';
        TripService.getPlanById(vm.planId).then(function (res) {
          vm.trip = res.data;
          vm.hasUnlocatedStops = hasUnlocatedStops(vm.trip);
          if (vm.trip.dayPlans && vm.trip.dayPlans.length && !vm.selectedDayNumber) vm.selectedDayNumber = vm.trip.dayPlans[0].dayNumber;
        }).catch(function (err) {
          vm.error = readError(err, 'Failed to load plan data');
        });
      };

      vm.loadAdminStatus();
      vm.loadTrip();
    }]);

  function readAdminFlag($window) {
    var enabled = readAdminFromURL($window.location);
    if (enabled) {
      clearViewerForce($window);
    }
    return enabled;
  }

  function readAdminFromURL(loc) {
    try {
      var searchParams = new URLSearchParams(loc.search || '');
      if (searchParams.get('admin') === 'true') return true;

      var hash = String(loc.hash || '');
      var qIndex = hash.indexOf('?');
      if (qIndex >= 0) {
        var hashQuery = hash.slice(qIndex + 1);
        var hashParams = new URLSearchParams(hashQuery);
        if (hashParams.get('admin') === 'true') return true;
      }
    } catch (e) {}
    return false;
  }

  function isViewerForced($window) {
    try {
      return $window.sessionStorage.getItem('forceViewerMode') === '1';
    } catch (e) {
      return false;
    }
  }

  function forceViewerMode($window) {
    try {
      $window.sessionStorage.setItem('forceViewerMode', '1');
    } catch (e) {}
  }

  function clearViewerForce($window) {
    try {
      $window.sessionStorage.removeItem('forceViewerMode');
    } catch (e) {}
  }

  function dayColor(dayNumber) {
    var colors = ['#e63946', '#1d3557', '#2a9d8f', '#f4a261', '#3a86ff', '#8e44ad'];
    return colors[(dayNumber - 1) % colors.length];
  }

  function hasUnlocatedStops(trip) {
    if (!trip || !trip.dayPlans) return false;
    return trip.dayPlans.some(function (day) {
      return (day.stops || []).some(function (stop) {
        return Number(stop.latitude) === 0 && Number(stop.longitude) === 0;
      });
    });
  }

  function findDay(trip, dayNumber) {
    var dayPlans = (trip && trip.dayPlans) || [];
    for (var i = 0; i < dayPlans.length; i++) {
      if (Number(dayPlans[i].dayNumber) === Number(dayNumber)) return dayPlans[i];
    }
    return null;
  }

  function scrollAgendaToSelection(dayNumber, stopId) {
    var retries = 8;
    var attempt = function () {
      var pane = document.querySelector('.side-pane');
      var target = null;
      if (stopId) target = document.getElementById('stop-item-' + stopId);
      if (!target) target = document.getElementById('day-block-' + dayNumber);
      if (pane && target) {
        var paneRect = pane.getBoundingClientRect();
        var targetRect = target.getBoundingClientRect();
        var delta = (targetRect.top - paneRect.top) - (pane.clientHeight / 2) + (targetRect.height / 2);
        pane.scrollTo({ top: pane.scrollTop + delta, behavior: 'smooth' });
        return;
      }
      retries -= 1;
      if (retries > 0) $timeout(attempt, 70);
    };
    $timeout(attempt, 0);
  }

  function parseISODate(value) {
    if (!value) return null;
    var m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(value));
    if (!m) return null;
    return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
  }

  function formatISODate(value) {
    if (!value) return '';
    var d = value instanceof Date ? value : new Date(value);
    if (!d || isNaN(d.getTime())) return '';
    var y = d.getFullYear();
    var m = String(d.getMonth() + 1).padStart(2, '0');
    var day = String(d.getDate()).padStart(2, '0');
    return y + '-' + m + '-' + day;
  }

  function splitImageUrls(text) {
    if (!text) return ['https://via.placeholder.com/320x180?text=Stop'];
    return text.split(',').map(function (item) { return item.trim(); }).filter(Boolean);
  }

  function readError(err, fallback) {
    if (err && err.data && err.data.error) return err.data.error;
    return fallback;
  }
})();
