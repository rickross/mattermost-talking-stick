import SidebarRight from './components/sidebar_right';
import TalkingStickIcon from './components/icon';
import Reducer from './reducers';
import {handleQueueUpdate} from './websocket';
import manifest from './manifest';

const {id: pluginId} = manifest;

class PluginClass {
    async initialize(registry, store) {
        // Register reducer
        registry.registerReducer(Reducer);

        // Register RHS panel
        const {showRHSPlugin} = registry.registerRightHandSidebarComponent(
            SidebarRight,
            'Talking Stick'
        );

        // Register channel header button
        registry.registerChannelHeaderButtonAction(
            TalkingStickIcon,
            () => store.dispatch(showRHSPlugin),
            'Talking Stick Queue',
            'Talking Stick Queue'
        );

        // Register WebSocket event handlers
        registry.registerWebSocketEventHandler(
            `custom_${pluginId}_queue_updated`,
            handleQueueUpdate(store)
        );
    }

    deinitialize() {
        // Cleanup if needed
    }
}

global.window.registerPlugin(pluginId, new PluginClass());
