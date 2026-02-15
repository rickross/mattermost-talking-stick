import {combineReducers} from 'redux';

const initialState = {
    queue: [],
    currentSpeaker: null,
    metrics: {},
};

function talkingStick(state = initialState, action) {
    switch (action.type) {
    case 'QUEUE_UPDATED':
        return {...state, queue: action.data};
    case 'SPEAKER_CHANGED':
        return {...state, currentSpeaker: action.data};
    case 'METRICS_UPDATED':
        return {...state, metrics: action.data};
    default:
        return state;
    }
}

export default combineReducers({
    talkingStick,
});
