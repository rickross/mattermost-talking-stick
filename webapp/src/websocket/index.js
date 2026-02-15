import {queueUpdated, speakerChanged} from '../actions';

export function handleQueueUpdate(store) {
    return (event) => {
        const data = JSON.parse(event.data);
        store.dispatch(queueUpdated(data.queue));
        if (data.currentSpeaker) {
            store.dispatch(speakerChanged(data.currentSpeaker));
        }
    };
}
