(function () {
  'use strict';

  angular.module('travelPlannerApp').directive('leafletMap', ['$timeout', function ($timeout) {
    return {
      restrict: 'E',
      scope: {
        trip: '=',
        isAdmin: '=',
        activeDayNumber: '=',
        activeStopId: '=',
        focusRequest: '=',
        addPointMode: '=',
        editDraft: '=',
        onMapClick: '&',
        onEditMarkerMoved: '&',
        onEdit: '&',
        onDelete: '&',
        dayColor: '&'
      },
      template: '<div class="map-canvas"></div>',
      link: function (scope, element) {
        var mapNode = element[0].querySelector('.map-canvas');
        var map = L.map(mapNode).setView([31.2304, 121.4737], 12);
        var layers = [];
        var dayLines = {};
        var stopMarkers = {};
        var dayBounds = {};
        var allBounds = [];
        var editMarker = null;

        var gaodeNormal = L.tileLayer(
          'https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}',
          {
            subdomains: ['1', '2', '3', '4'],
            attribution: '&copy; AutoNavi'
          }
        );
        var gaodeSatellite = L.tileLayer(
          'https://webst0{s}.is.autonavi.com/appmaptile?style=6&x={x}&y={y}&z={z}',
          {
            subdomains: ['1', '2', '3', '4'],
            attribution: '&copy; AutoNavi'
          }
        );
        var gaodeAnnotation = L.tileLayer(
          'https://webst0{s}.is.autonavi.com/appmaptile?style=8&x={x}&y={y}&z={z}',
          {
            subdomains: ['1', '2', '3', '4'],
            attribution: '&copy; AutoNavi'
          }
        );
        var osm = L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
          attribution: '&copy; OpenStreetMap contributors'
        });
        var openTopoMap = L.tileLayer('https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png', {
          maxZoom: 17,
          attribution: '&copy; OpenTopoMap contributors, &copy; OpenStreetMap contributors'
        });
        var cartoLight = L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png', {
          attribution: '&copy; CARTO, &copy; OpenStreetMap contributors'
        });
        var cartoDark = L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
          attribution: '&copy; CARTO, &copy; OpenStreetMap contributors'
        });
        var esriImagery = L.tileLayer(
          'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
          {
            attribution: 'Tiles &copy; Esri'
          }
        );
        var esriTopo = L.tileLayer(
          'https://server.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}',
          {
            attribution: 'Tiles &copy; Esri'
          }
        );

        osm.addTo(map);
        L.control.layers(
          {
            'Gaode Street': gaodeNormal,
            'Gaode Satellite': gaodeSatellite,
            'OpenStreetMap': osm,
            'OpenTopoMap': openTopoMap,
            'Esri World Imagery': esriImagery,
            'Esri Topographic': esriTopo,
            'CARTO Light': cartoLight,
            'CARTO Dark': cartoDark
          },
          {
            'Gaode Labels': gaodeAnnotation
          },
          { position: 'topright' }
        ).addTo(map);
        L.control.scale({ imperial: false, position: 'bottomright' }).addTo(map);

        scope.$watch('trip', function (trip) {
          render(trip);
        }, true);
        scope.$watch('activeDayNumber', function () {
          updateDayHighlight();
        });
        scope.$watch('focusRequest', function (request) {
          if (!request) return;
          handleFocusRequest(request);
        }, true);
        scope.$watch('editDraft', function (draft) {
          syncEditMarker(draft);
        }, true);

        map.on('click', function (evt) {
          if (!scope.addPointMode) return;
          scope.onMapClick({ payload: { lat: evt.latlng.lat, lng: evt.latlng.lng } });
          scope.$applyAsync();
        });

        function render(trip) {
          clearLayers();
          if (!trip || !trip.dayPlans || !trip.dayPlans.length) return;

          var bounds = [];
          allBounds = [];

          trip.dayPlans.forEach(function (dayPlan) {
            var color = scope.dayColor({ dayNumber: dayPlan.dayNumber });
            var latLngs = [];
            dayBounds[dayPlan.dayNumber] = [];

            dayPlan.stops.forEach(function (stop) {
              var unresolved = Number(stop.latitude) === 0 && Number(stop.longitude) === 0;
              var latLng = unresolved ? [map.getCenter().lat, map.getCenter().lng] : [stop.latitude, stop.longitude];
              if (!unresolved) {
                latLngs.push(latLng);
                bounds.push(latLng);
                allBounds.push(latLng);
                dayBounds[dayPlan.dayNumber].push(latLng);
              }

              var marker = L.marker(latLng, { icon: unresolved ? buildUnknownMarkerIcon() : buildMarkerIcon(dayPlan.dayNumber, stop.name, color) });
              marker.bindPopup(buildPopupHtml(dayPlan.dayNumber, stop, scope.isAdmin));
              marker.on('popupopen', function () {
                wirePopupActions(dayPlan.dayNumber, stop);
              });
              marker.on('click', function () {
                scope.activeDayNumber = dayPlan.dayNumber;
                scope.activeStopId = stop.id;
                scope.$applyAsync();
              });
              marker.addTo(map);
              layers.push(marker);
              stopMarkers[stop.id] = marker;
            });

            if (latLngs.length > 1) {
              var line = L.polyline(latLngs, { color: color, weight: 4, opacity: 0.85 }).addTo(map);
              layers.push(line);
              dayLines[dayPlan.dayNumber] = line;
            }
          });

          if (bounds.length) {
            $timeout(function () {
              map.invalidateSize();
              map.fitBounds(bounds, { padding: [24, 24] });
            }, 0);
          }
          updateDayHighlight();
        }

        function clearLayers() {
          layers.forEach(function (layer) {
            map.removeLayer(layer);
          });
          layers = [];
          dayLines = {};
          stopMarkers = {};
          dayBounds = {};
          allBounds = [];
        }

        function updateDayHighlight() {
          Object.keys(dayLines).forEach(function (key) {
            var day = Number(key);
            var isActive = day === Number(scope.activeDayNumber);
            dayLines[day].setStyle({
              weight: isActive ? 7 : 4,
              opacity: isActive ? 1 : 0.65
            });
          });
        }

        function handleFocusRequest(request) {
          if (request.mode === 'day' && dayBounds[request.dayNumber] && dayBounds[request.dayNumber].length) {
            map.fitBounds(dayBounds[request.dayNumber], { padding: [24, 24] });
            updateDayHighlight();
            return;
          }
          if (request.mode === 'stop' && stopMarkers[request.stopId]) {
            map.setView(stopMarkers[request.stopId].getLatLng(), 14);
            stopMarkers[request.stopId].openPopup();
            updateDayHighlight();
            return;
          }
          if (request.mode === 'all' && allBounds.length) {
            map.fitBounds(allBounds, { padding: [24, 24] });
            updateDayHighlight();
          }
        }

        function syncEditMarker(draft) {
          if (!draft) {
            if (editMarker) {
              map.removeLayer(editMarker);
              editMarker = null;
            }
            return;
          }
          var latLng = [Number(draft.latitude), Number(draft.longitude)];
          if (!editMarker) {
            editMarker = L.marker(latLng, { draggable: true }).addTo(map);
            editMarker.on('dragend', function () {
              var p = editMarker.getLatLng();
              scope.onEditMarkerMoved({ payload: { lat: p.lat, lng: p.lng } });
              scope.$applyAsync();
            });
          } else {
            editMarker.setLatLng(latLng);
          }
          map.setView(latLng, 14);
        }

        function buildPopupHtml(dayNumber, stop, isAdmin) {
          var image = (stop.imageUrls && stop.imageUrls[0]) || 'https://via.placeholder.com/320x180?text=Stop';
          var html = ''
            + '<div class="popup-content">'
            + '<h4>' + escapeHtml(stop.name) + '</h4>'
            + '<p><strong>Day ' + dayNumber + '</strong></p>'
            + '<p>' + escapeHtml(stop.activityDescription || '') + '</p>'
            + '<a href="' + escapeHtml(image) + '" target="_blank" rel="noreferrer">'
            + '<img class="popup-thumb" src="' + escapeHtml(image) + '" alt="thumbnail" />'
            + '</a>';

          if (isAdmin) {
            html += ''
              + '<div class="popup-actions">'
              + '<button type="button" class="popup-edit" data-stop-id="' + escapeHtml(stop.id) + '" data-day="' + dayNumber + '">编辑</button>'
              + '<button type="button" class="popup-delete" data-stop-id="' + escapeHtml(stop.id) + '">删除</button>'
              + '</div>';
          }

          html += '</div>';
          return html;
        }

        function wirePopupActions(dayNumber, stop) {
          var editBtn = document.querySelector('.popup-edit[data-stop-id="' + cssEscape(stop.id) + '"]');
          var delBtn = document.querySelector('.popup-delete[data-stop-id="' + cssEscape(stop.id) + '"]');

          if (editBtn) {
            editBtn.onclick = function () {
              scope.onEdit({ payload: { dayNumber: dayNumber, stop: stop } });
              scope.$applyAsync();
            };
          }

          if (delBtn) {
            delBtn.onclick = function () {
              scope.onDelete({ payload: { dayNumber: dayNumber, stop: stop } });
              scope.$applyAsync();
            };
          }
        }

        function buildMarkerIcon(dayNumber, stopName, color) {
          var shortName = abbreviate(stopName);
          var html = '<div class="stop-marker" style="background:' + color + ';">D' + dayNumber + '·' + shortName + '</div>';
          return L.divIcon({
            className: 'custom-stop-wrapper',
            html: html,
            iconSize: [72, 24],
            iconAnchor: [36, 24]
          });
        }

        function buildUnknownMarkerIcon() {
          var html = '<div class="stop-marker" style="background:#6b7280;">?</div>';
          return L.divIcon({
            className: 'custom-stop-wrapper',
            html: html,
            iconSize: [28, 24],
            iconAnchor: [14, 24]
          });
        }

        function abbreviate(name) {
          if (!name) return 'SP';
          var words = name.trim().split(/\s+/);
          if (words.length === 1) return words[0].slice(0, 3).toUpperCase();
          return (words[0][0] + words[1][0]).toUpperCase();
        }

        function escapeHtml(text) {
          return String(text)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#39;');
        }

        function cssEscape(value) {
          return String(value).replace(/"/g, '\\"');
        }
      }
    };
  }]);
})();
