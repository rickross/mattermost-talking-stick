export function queueUpdated(queue) {
    return {
        type: 'QUEUE_UPDATED',
        data: queue,
    };
}

export function speakerChanged(speaker) {
    return {
        type: 'SPEAKER_CHANGED',
        data: speaker,
    };
}

export function metricsUpdated(metrics) {
    return {
        type: 'METRICS_UPDATED',
        data: metrics,
    };
}
