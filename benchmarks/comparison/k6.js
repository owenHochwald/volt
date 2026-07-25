import http from "k6/http";

export const options = {
  vus: Number(__ENV.VUS),
  duration: __ENV.DURATION,
  discardResponseBodies: false,
  noConnectionReuse: false,
  noVUConnectionReuse: false,
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "p(99)", "min", "max"],
};

export default function () {
  http.get(__ENV.TARGET_URL, {
    headers: {
      "Accept-Encoding": "identity",
    },
  });
}

export function handleSummary(data) {
  const requests = data.metrics.http_reqs.values;
  return {
    [__ENV.SUMMARY_PATH]: JSON.stringify(
      {
        requests: requests.count,
        requests_per_second: requests.rate,
      },
      null,
      2,
    ),
  };
}
