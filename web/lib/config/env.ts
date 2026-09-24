export function mercuryAPIURL(): string {
  const configuredURL = process.env.MERCURY_API_URL;
  if (configuredURL === undefined) {
    return "http://localhost:8080";
  }

  let parsedURL: URL;
  try {
    parsedURL = new URL(configuredURL);
  } catch {
    throw new Error("MERCURY_API_URL must be a valid absolute URL");
  }
  if (!["http:", "https:"].includes(parsedURL.protocol)) {
    throw new Error("MERCURY_API_URL must use http or https");
  }
  if (parsedURL.username || parsedURL.password) {
    throw new Error("MERCURY_API_URL must not contain credentials");
  }

  return parsedURL.toString().replace(/\/$/, "");
}
