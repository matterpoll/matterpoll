import {AnyAction, Dispatch} from 'redux';

import {getCurrentTeamId} from 'mattermost-redux/selectors/entities/teams';
import {GetStateFunc} from 'mattermost-redux/types/actions';
import {Client4} from 'mattermost-redux/client';
import {IntegrationTypes} from 'mattermost-redux/action_types';

export async function clientExecuteCommand(dispatch: Dispatch<AnyAction>, getState: GetStateFunc, command: string, teamId: string, channelId: string, rootId: string) {
    const args = {
        channel_id: channelId,
        team_id: teamId || getCurrentTeamId(getState()),
        root_id: rootId,
    };

    try {
        //@ts-ignore Typing in mattermost-redux is wrong
        const data = await Client4.executeCommand(command, args);
        dispatch(setTriggerId(data?.trigger_id));
    } catch (error) {
        console.error(error); //eslint-disable-line no-console
    }
}

export function setTriggerId(triggerId: string) {
    return {
        type: IntegrationTypes.RECEIVED_DIALOG_TRIGGER_ID,
        data: triggerId,
    };
}
