<script>
  import CalHeatmap from "cal-heatmap";
  import CalendarLabel from "cal-heatmap/plugins/CalendarLabel";
  import Legend from "cal-heatmap/plugins/Legend";
  import Tooltip from "cal-heatmap/plugins/Tooltip";
  import { onDestroy } from "svelte";

  import "cal-heatmap/cal-heatmap.css";
  import dayjs from "dayjs";

  var localeData = require("dayjs/plugin/localeData");
  dayjs.extend(localeData);

  dayjs().localeData();

  function getScreenSize() {
    const breakpoints = {
      sm: 640,
      md: 768,
      lg: 1024,
      xl: 1280,
      "2xl": 1536,
    };
    const width = window.innerWidth;

    if (width < breakpoints.sm) {
      return "small";
    } else if (width < breakpoints.md) {
      return "medium";
    } else if (width < breakpoints.lg) {
      return "large";
    } else {
      return "extralarge";
    }
  }

  let screenSize = getScreenSize();
  let size = {
    small: {
      subtract: 3,
      range: 4,
    },
    medium: {
      subtract: 4,
      range: 5,
    },
    large: {
      subtract: 5,
      range: 6,
    },
    extralarge: {
      subtract: 6,
      range: 7,
    },
  };
  let currentSize = size[screenSize];

  export let countsByDay = [
    { date: "2024-01-01T00:00:00Z", value: 1 },
    { date: "2024-03-02T00:00:00Z", value: 6 },
    { date: "2024-04-02T00:00:00Z", value: 16 },
    { date: "2024-04-03T00:00:00Z", value: 9 },
    { date: "2024-04-04T00:00:00Z", value: 9 },
  ];

  function cleanUpCountsByDay(counts) {
    return counts.map((item) => ({
      date: dayjs(item.date).format("YYYY-MM-DD"),
      value: item.value,
    }));
  }

  countsByDay = cleanUpCountsByDay(countsByDay);

  let start = dayjs().subtract(currentSize.subtract, "month");

  const upToTodayTemplate = function (DateHelper) {
    return {
      name: "ghDayUpToToday",
      parent: "ghDay",

      mapping: (startTimestamp, endTimestamp) => {
        let weekNumber = 0;
        let x = -1;

        return DateHelper.intervals(
          "day",
          startTimestamp,
          DateHelper.date(endTimestamp)
        )
          .map((ts) => {
            const date = DateHelper.date(ts);

            today = new Date();
            if (date > today) {
              return null;
            }

            if (weekNumber !== date.week()) {
              weekNumber = date.week();
              x += 1;
            }

            return {
              t: ts,
              x,
              y: date.day(),
            };
          })
          .filter((n) => n !== null);
      },
    };
  };

  const cal = new CalHeatmap();
  cal.addTemplates(upToTodayTemplate);

  function paintHeatmap(theme) {
    cal.paint(
      {
        data: { source: countsByDay, x: "date", y: "value" },
        range: currentSize.range,
        date: { start: start, max: new Date() },
        domain: {
          type: "month",
          gutter: 10,
          dynamicDimension: true,
          label: { position: "top" },
        },
        subDomain: {
          type: "ghDayUpToToday",
          radius: 3,
          width: 15,
          height: 15,
          gutter: 4,
        },
        scale: {
          color: {
            type: "linear",
            // More schemes at https://observablehq.com/@d3/color-schemes
            scheme: "Greens",
            domain: [0, 3],
          },
        },

        theme: theme,
      },
      [
        [
          Tooltip,
          {
            text: function (date, value, dayjsDate) {
              return (
                (value ? value : "No") +
                " interactions on " +
                dayjsDate.format("dddd, MMMM D, YYYY")
              );
            },
          },
        ],
        [
          Legend,
          {
            tickSize: 0,
            width: 100,
            itemSelector: "#legend",
            label: "",
          },
        ],
        [
          CalendarLabel,
          {
            width: 30,
            padding: [24, 0, 0, 0],
            textAlign: "start",
            text: () =>
              dayjs.weekdaysShort().map((d, i) => (i % 2 == 0 ? "" : d)),
          },
        ],
      ]
    );
  }

  function getLightOrDarkTheme() {
    let siteTheme = document.documentElement.getAttribute("data-theme");
    if (siteTheme == "lightmode") {
      return "light";
    } else {
      return "dark";
    }
  }

  paintHeatmap(getLightOrDarkTheme());

  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (
        mutation.type === "attributes" &&
        mutation.attributeName === "data-theme"
      ) {
        // Update the data-theme attribute on the first svg child of the heatmap div
        const svgElement = document.querySelector("#cal-heatmap > svg");
        if (svgElement) {
          svgElement.setAttribute("data-theme", getLightOrDarkTheme());
        }
      }
    });
  });

  observer.observe(document.documentElement, {
    attributes: true,
  });

  // Cleanup observer on component destroy
  onDestroy(() => {
    observer.disconnect();
  });
</script>

<div class=" w-screen md:w-full md:overflow-hidden">
  <div
    id="horizontal-scroll-container"
    class="flex flex-col justify-center items-center p-4 w-full overflow-x-auto h-min"
  >
    <div id="cal-heatmap" class="m-2 p-5"></div>
    <!-- <div id="legend"></div> -->
  </div>
</div>
