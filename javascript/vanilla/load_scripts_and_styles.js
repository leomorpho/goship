export function loadScriptsAndStyles(files) {
  // Promise array to track loading status
  let promises = [];

  // Helper function to create a promise for each file
  function createPromise(file) {
    return new Promise((resolve, reject) => {
      let element;

      if (file.endsWith(".js")) {
        // Create script element for JavaScript file
        element = document.createElement("script");
        element.src = file;
        element.onload = resolve;
        element.onerror = reject;
        document.head.appendChild(element);
      } else if (file.endsWith(".css")) {
        // Create link element for CSS file
        element = document.createElement("link");
        element.rel = "stylesheet";
        element.href = file;
        element.onload = resolve;
        element.onerror = reject;
        document.head.appendChild(element);
      } else {
        reject(new Error(`Unsupported file type: ${file}`));
      }
    });
  }

  // Create promises for each file and push them to the promises array
  files.forEach((file) => {
    promises.push(createPromise(file));
  });

  // Return a promise that resolves when all promises are resolved
  return Promise.all(promises);
}
