export function createCalHeatmap(countsByDay, containerId = "cal-heatmap") {
  // Check if a heatmap already exists in the container
  if (document.querySelector(`#${containerId} > svg`)) {
    console.log("Heatmap already exists. Skipping creation.");
    return; // Exit early if a heatmap is already present
  }

  // Import necessary dependencies (assume they are already loaded globally)
  // Usage example
  let filesToLoad = [
    { src: "https://cdn.jsdelivr.net/npm/dayjs@1/dayjs.min.js", type: "js" },
    { src: "https://d3js.org/d3.v7.min.js", type: "js" },
    {
      src: "https://unpkg.com/cal-heatmap/dist/cal-heatmap.min.js",
      type: "js",
    },
    {
      src: "https://unpkg.com/cal-heatmap/dist/cal-heatmap.css",
      type: "style",
    },
    {
      src: "https://unpkg.com/cal-heatmap/dist/plugins/Legend.min.js",
      type: "js",
    },
    { src: "https://unpkg.com/@popperjs/core@2", type: "js" },
    {
      src: "https://unpkg.com/cal-heatmap/dist/plugins/Tooltip.min.js",
      type: "js",
    },
    {
      src: "https://unpkg.com/cal-heatmap/dist/plugins/CalendarLabel.min.js",
      type: "js",
    },
  ];

  function loadScriptsAndStyles(files) {
    let promises = [];

    function createPromise(file) {
      return new Promise((resolve, reject) => {
        let element;

        if (file.type === "js") {
          element = document.createElement("script");
          element.src = file.src;
          element.onload = resolve;
          element.onerror = reject;
          document.head.appendChild(element);
        } else if (file.type === "style") {
          element = document.createElement("link");
          element.rel = "stylesheet";
          element.href = file.src;
          element.onload = resolve;
          element.onerror = reject;
          document.head.appendChild(element);
        } else {
          reject(new Error(`Unsupported file type: ${file.type}`));
        }
      });
    }

    files.forEach((file) => {
      promises.push(createPromise(file));
    });

    return Promise.all(promises);
  }

  // Usage example
  loadScriptsAndStyles(filesToLoad)
    .then(() => {
      console.log("All files loaded successfully");
      // Call your function here
      createHeatmap(countsByDay, containerId);
    })
    .catch((error) => {
      console.error("Error loading files:", error);
    });
}

function createHeatmap(countsByDay, containerId = "cal-heatmap") {
  // Import necessary dependencies (assume they are already loaded globally)

  // Clean up countsByDay data
  function cleanUpCountsByDay(counts) {
    return counts.map((item) => ({
      date: dayjs(item.date).format("YYYY-MM-DD"),
      value: item.value,
    }));
  }

  countsByDay = cleanUpCountsByDay(countsByDay);

  // Determine screen size
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

  let start = dayjs().subtract(currentSize.subtract, "month");

  // Define the upToTodayTemplate function
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

            let today = new Date();
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

  // Create CalHeatmap instance
  const cal = new CalHeatmap();
  cal.addTemplates(upToTodayTemplate);

  // Function to paint the heatmap
  function paintHeatmap(theme) {
    cal.paint({
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
          scheme: "Greens",
          domain: [0, 3],
        },
      },
      theme: theme,
    });
  }

  // Initialize the heatmap
  paintHeatmap(getLightOrDarkTheme());

  // Function to get light or dark theme based on data-theme attribute
  function getLightOrDarkTheme() {
    let siteTheme = document.documentElement.getAttribute("data-theme");
    if (siteTheme == "lightmode") {
      return "light";
    } else {
      return "dark";
    }
  }

  // Mutation observer to detect changes in data-theme attribute
  const observer = new MutationObserver((mutations) => {
    mutations.forEach((mutation) => {
      if (
        mutation.type === "attributes" &&
        mutation.attributeName === "data-theme"
      ) {
        // Update the data-theme attribute on the first svg child of the heatmap div
        const svgElement = document.querySelector(`#${containerId} > svg`);
        if (svgElement) {
          svgElement.setAttribute("data-theme", getLightOrDarkTheme());
        }
      }
    });
  });

  // Observe changes in data-theme attribute
  observer.observe(document.documentElement, {
    attributes: true,
  });

  // Return the observer so it can be disconnected externally if needed
  return observer;
}
