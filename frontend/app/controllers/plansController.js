(function () {
  'use strict';

  angular.module('travelPlannerApp')
    .controller('PlansController', ['TripService', '$window', '$location', '$q', '$timeout', function (TripService, $window, $location, $q, $timeout) {
      var vm = this;
      vm.plans = [];
      vm.filteredPlans = [];
      vm.error = '';
      vm.isAdmin = false;
      vm.showCreate = false;
      vm.showEdit = false;
      vm.createForm = emptyPlanForm();
      vm.editForm = emptyPlanForm();
      vm.editPlanId = '';
      vm.searchText = '';
      vm.sortOrder = 'dateAsc';
      vm.importModal = {
        show: false,
        planName: '',
        file: null,
        fileName: '',
        dragActive: false,
        uploading: false,
        progress: 0,
        error: '',
        warnings: [],
        result: null
      };

      vm.loadAdminStatus = function () {
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

      vm.loadPlans = function () {
        TripService.getAllPlans().then(function (res) {
          var list = res.data.plans || [];
          var jobs = list.map(function (plan) {
            return TripService.getPlanById(plan.id).then(function (detailRes) {
              var detail = detailRes.data;
              plan.stopCount = countStops(detail.dayPlans || []);
              plan.dayCount = calcDayCount(plan.startDate, plan.endDate);
              plan.coverImage = resolveCoverImage(plan, detail);
              return plan;
            }).catch(function () {
              plan.stopCount = 0;
              plan.dayCount = calcDayCount(plan.startDate, plan.endDate);
              plan.coverImage = plan.coverImage || defaultCover(plan.name);
              return plan;
            });
          });
          return $q.all(jobs);
        }).then(function (plans) {
          vm.plans = plans;
          vm.applyFilters();
        }).catch(function (err) {
          vm.error = readError(err, 'Failed to load plans');
        });
      };

      vm.applyFilters = function () {
        var keyword = (vm.searchText || '').trim().toLowerCase();
        var out = vm.plans.filter(function (p) {
          return !keyword || (p.name || '').toLowerCase().indexOf(keyword) >= 0;
        });
        out.sort(function (a, b) {
          var da = new Date(a.startDate).getTime() || 0;
          var db = new Date(b.startDate).getTime() || 0;
          return vm.sortOrder === 'dateDesc' ? db - da : da - db;
        });
        vm.filteredPlans = out;
      };

      vm.openPlan = function (plan) {
        $location.path('/plan/' + encodeURIComponent(plan.id));
      };

      vm.openCreateModal = function () {
        if (!vm.isAdmin) return;
        vm.createForm = emptyPlanForm();
        vm.showCreate = true;
      };
      vm.closeCreateModal = function () { vm.showCreate = false; };

      vm.submitCreate = function () {
        if (!vm.isAdmin) return;
        var payload = {
          name: vm.createForm.name,
          startDate: vm.createForm.startDate,
          endDate: vm.createForm.endDate,
          coverImage: vm.createForm.coverImage || defaultCover(vm.createForm.name),
          dayPlans: []
        };
        TripService.createPlan(payload, true).then(function () {
          vm.showCreate = false;
          vm.loadPlans();
        }).catch(function (err) {
          vm.error = readError(err, 'Create plan failed');
        });
      };

      vm.openEditModal = function (plan, $event) {
        if ($event) $event.stopPropagation();
        if (!vm.isAdmin) return;
        vm.editPlanId = plan.id;
        vm.editForm = {
          name: plan.name,
          startDate: plan.startDate,
          endDate: plan.endDate,
          coverImage: plan.coverImage
        };
        vm.showEdit = true;
      };
      vm.closeEditModal = function () { vm.showEdit = false; };

      vm.submitEdit = function () {
        if (!vm.isAdmin) return;
        TripService.getPlanById(vm.editPlanId).then(function (res) {
          var planDetail = res.data;
          planDetail.name = vm.editForm.name;
          planDetail.startDate = vm.editForm.startDate;
          planDetail.endDate = vm.editForm.endDate;
          planDetail.coverImage = vm.editForm.coverImage || defaultCover(vm.editForm.name);
          return TripService.updatePlan(vm.editPlanId, planDetail, true);
        }).then(function () {
          vm.showEdit = false;
          vm.loadPlans();
        }).catch(function (err) {
          vm.error = readError(err, 'Update plan failed');
        });
      };

      vm.deletePlan = function (plan, $event) {
        if ($event) $event.stopPropagation();
        if (!vm.isAdmin) return;
        if (!$window.confirm('Delete this plan?')) return;
        TripService.deletePlan(plan.id, true).then(function () {
          vm.loadPlans();
        }).catch(function (err) {
          vm.error = readError(err, 'Delete plan failed');
        });
      };

      vm.copyPlan = function (plan, $event) {
        if ($event) $event.stopPropagation();
        if (!vm.isAdmin) return;
        TripService.getPlanById(plan.id).then(function (res) {
          var src = res.data;
          var payload = {
            name: src.name + '（副本）',
            startDate: src.startDate,
            endDate: src.endDate,
            coverImage: src.coverImage || defaultCover(src.name),
            dayPlans: src.dayPlans || []
          };
          return TripService.createPlan(payload, true);
        }).then(function () {
          vm.loadPlans();
        }).catch(function (err) {
          vm.error = readError(err, 'Copy plan failed');
        });
      };

      vm.downloadTemplate = function () {
        if (!vm.isAdmin) return;
        var url = TripService.getImportTemplateURL(true);
        $window.open(url, '_blank');
      };

      vm.openImportModal = function () {
        if (!vm.isAdmin) return;
        vm.importModal = {
          show: true,
          planName: '',
          file: null,
          fileName: '',
          dragActive: false,
          uploading: false,
          progress: 0,
          error: '',
          warnings: [],
          result: null
        };
        $timeout(bindDropZone, 0);
      };

      vm.closeImportModal = function () {
        vm.importModal.show = false;
      };

      vm.pickExcelFile = function () {
        var input = document.getElementById('excelImportInput');
        if (input) input.click();
      };

      vm.onFileInputChange = function (evt) {
        var f = evt.target && evt.target.files && evt.target.files[0];
        vm.setImportFile(f);
      };

      vm.setImportFile = function (file) {
        vm.importModal.error = '';
        if (!file) return;
        var validateErr = validateExcelFile(file);
        if (validateErr) {
          vm.importModal.file = null;
          vm.importModal.fileName = '';
          vm.importModal.error = validateErr;
          return;
        }
        vm.importModal.file = file;
        vm.importModal.fileName = file.name;
      };

      vm.uploadExcel = function () {
        if (!vm.isAdmin) return;
        vm.importModal.error = '';
        if (!vm.importModal.file) {
          vm.importModal.error = '请先选择 Excel 文件';
          return;
        }
        vm.importModal.uploading = true;
        vm.importModal.progress = 0;
        vm.importModal.warnings = [];
        vm.importModal.result = null;

        TripService.importPlan(
          vm.importModal.file,
          vm.importModal.planName,
          true,
          function (percent) {
            vm.importModal.progress = percent;
          }
        ).then(function (res) {
          vm.handleImportSuccess(res.data);
        }).catch(function (err) {
          vm.importModal.error = readError(err, '导入失败');
        }).finally(function () {
          vm.importModal.uploading = false;
          if (vm.importModal.progress < 100 && !vm.importModal.error) vm.importModal.progress = 100;
        });
      };

      vm.handleImportSuccess = function (data) {
        vm.importModal.warnings = (data && data.warnings) || [];
        vm.importModal.result = data || null;
        vm.importModal.progress = 100;
        vm.loadPlans();
      };

      vm.viewImportedPlan = function () {
        if (!vm.importModal.result || !vm.importModal.result.planId) return;
        vm.closeImportModal();
        $location.path('/plan/' + encodeURIComponent(vm.importModal.result.planId));
      };

      function bindDropZone() {
        var zone = document.getElementById('excelDropZone');
        if (!zone) return;
        if (zone.dataset.bound === '1') return;
        zone.dataset.bound = '1';

        zone.addEventListener('dragover', function (evt) {
          evt.preventDefault();
          vm.importModal.dragActive = true;
          $timeout(angular.noop, 0);
        });
        zone.addEventListener('dragleave', function (evt) {
          evt.preventDefault();
          vm.importModal.dragActive = false;
          $timeout(angular.noop, 0);
        });
        zone.addEventListener('drop', function (evt) {
          evt.preventDefault();
          vm.importModal.dragActive = false;
          var file = evt.dataTransfer && evt.dataTransfer.files && evt.dataTransfer.files[0];
          vm.setImportFile(file);
          $timeout(angular.noop, 0);
        });
      }

      vm.loadAdminStatus();
      vm.loadPlans();
    }]);

  function emptyPlanForm() {
    return {
      name: '',
      startDate: '',
      endDate: '',
      coverImage: ''
    };
  }

  function calcDayCount(startDate, endDate) {
    var s = new Date(startDate);
    var e = new Date(endDate);
    var sv = s.getTime();
    var ev = e.getTime();
    if (!sv || !ev || ev < sv) return 0;
    return Math.floor((ev - sv) / 86400000) + 1;
  }

  function countStops(dayPlans) {
    return dayPlans.reduce(function (sum, day) {
      return sum + ((day.stops || []).length);
    }, 0);
  }

  function defaultCover(name) {
    var q = encodeURIComponent(name || 'travel');
    return 'https://source.unsplash.com/featured/640x360/?' + q + ',travel';
  }

  function resolveCoverImage(plan, detail) {
    if (plan.coverImage && plan.coverImage.indexOf('via.placeholder.com') === -1) {
      return plan.coverImage;
    }
    var points = collectCoords(detail && detail.dayPlans);
    if (!points.length) {
      return plan.coverImage || defaultCover(plan.name);
    }
    var c = centroid(points);
    var markerList = points.slice(0, 20).map(function (p) {
      return p.lat.toFixed(5) + ',' + p.lng.toFixed(5) + ',lightblue1';
    }).join('|');
    return 'https://staticmap.openstreetmap.de/staticmap.php?center='
      + c.lat.toFixed(5) + ',' + c.lng.toFixed(5)
      + '&zoom=7&size=640x360&markers=' + markerList;
  }

  function collectCoords(dayPlans) {
    var out = [];
    (dayPlans || []).forEach(function (day) {
      (day.stops || []).forEach(function (s) {
        var lat = Number(s.latitude);
        var lng = Number(s.longitude);
        if (!lat && !lng) return;
        if (!isFinite(lat) || !isFinite(lng)) return;
        out.push({ lat: lat, lng: lng });
      });
    });
    return out;
  }

  function centroid(points) {
    var sumLat = 0;
    var sumLng = 0;
    points.forEach(function (p) {
      sumLat += p.lat;
      sumLng += p.lng;
    });
    return { lat: sumLat / points.length, lng: sumLng / points.length };
  }

  function readAdminFlag($window) {
    var p = new URLSearchParams($window.location.search);
    var enabled = p.get('admin') === 'true';
    if (enabled) {
      clearViewerForce($window);
    }
    return enabled;
  }

  function isViewerForced($window) {
    try {
      return $window.sessionStorage.getItem('forceViewerMode') === '1';
    } catch (e) {
      return false;
    }
  }

  function clearViewerForce($window) {
    try {
      $window.sessionStorage.removeItem('forceViewerMode');
    } catch (e) {}
  }

  function readError(err, fallback) {
    if (err && err.data && err.data.error) return err.data.error;
    return fallback;
  }

  function validateExcelFile(file) {
    if (!file) return '请选择文件';
    var maxBytes = 10 * 1024 * 1024;
    if (file.size > maxBytes) return '文件过大，最大 10MB';

    var name = (file.name || '').toLowerCase();
    if (!(name.endsWith('.xls') || name.endsWith('.xlsx'))) {
      return '仅支持 .xls 或 .xlsx 文件';
    }
    return '';
  }
})();
