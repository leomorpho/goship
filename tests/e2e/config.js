const config = {
  development: {
    WEBSITE_URL: process.env.WEBSITE_URL || "http://localhost:8000",
    EMAIL_CAPTURE_PORTAL_URL: "http://localhost:8026",
  },
  production: {
    WEBSITE_URL: process.env.WEBSITE_URL || "https://yourproductionurl.com",
  },
  // other environments...
};

// Default configuration
const defaultConfig = {
  WEBSITE_URL: process.env.WEBSITE_URL || "http://localhost:8000",
};

module.exports = { config, defaultConfig };
