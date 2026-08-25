import React from 'react';
import {render, screen} from '@testing-library/react';

import {ActionButtonType} from '@/utils/constants';

import ActionView from '@/components/post_type/action_view/action_view';

import type {AttachmentAction, PollMetadata} from '@/types/poll';

// ActionButton is connected to the store; stand in for it so these tests cover
// ActionView's own filtering rather than the button's rendering.
jest.mock('@/components/post_type/action_view/action_button', () => ({
    __esModule: true,
    default: ({action, hasVoted}: {action: AttachmentAction; hasVoted?: boolean}) => (
        <button
            data-testid={action.id}
            data-has-voted={String(Boolean(hasVoted))}
        >
            {action.name}
        </button>
    ),
}));

describe('components/post_type/action_view/ActionView', () => {
    const samplePollId = 'samplepollid1';

    const baseMetadata: PollMetadata = {
        voted_answers: ['answer1', 'answer2'],
        poll_id: samplePollId,
        can_manage_poll: false,
        setting_progress: false,
        setting_public_add_option: false,
    };

    const baseProps = {
        post: {
            id: 'post_id',
            props: {
                poll_id: samplePollId,
            },
        },
        attachment: {
            actions: [
                {id: 'action_id1', name: 'answer1', type: ActionButtonType.BUTTON},
                {id: 'action_id2', name: 'answer2', type: ActionButtonType.BUTTON},
                {id: 'action_id3', name: 'answer3', type: ActionButtonType.BUTTON},
                {id: 'resetVote', name: 'Reset Your Vote', type: ActionButtonType.BUTTON},
                {id: 'addOption', name: 'Add option', type: ActionButtonType.BUTTON},
                {id: 'deletePoll', name: 'Delete Poll', type: ActionButtonType.BUTTON},
                {id: 'endPoll', name: 'End Poll', type: ActionButtonType.BUTTON},
            ],
        },
        pollMetadata: {[samplePollId]: baseMetadata},
        siteUrl: 'http://localhost:8065',
        actions: {
            fetchPollMetadata: jest.fn(),
        },
    };

    const renderedActionIds = () => screen.queryAllByRole('button').map((button) => button.dataset.testid);

    test('should fetch the poll metadata on mount', () => {
        const fetchPollMetadata = jest.fn();
        render(
            <ActionView
                {...baseProps}
                actions={{fetchPollMetadata}}
            />,
        );

        expect(fetchPollMetadata).toHaveBeenCalledWith('http://localhost:8065', samplePollId);
    });

    test('should hide the management and add-option buttons without permission', () => {
        render(<ActionView {...baseProps}/>);

        expect(renderedActionIds()).toEqual(['action_id1', 'action_id2', 'action_id3', 'resetVote']);
    });

    test('should render exactly one action container', () => {
        const {container} = render(<ActionView {...baseProps}/>);

        expect(container.querySelectorAll('.attachment-actions')).toHaveLength(1);
    });

    test('should show the management buttons with permission to manage the poll', () => {
        render(
            <ActionView
                {...baseProps}
                pollMetadata={{[samplePollId]: {...baseMetadata, can_manage_poll: true}}}
            />,
        );

        expect(renderedActionIds()).toContain('deletePoll');
        expect(renderedActionIds()).toContain('endPoll');
        expect(renderedActionIds()).toContain('addOption');
    });

    test('should show the add-option button when public-add-option is set', () => {
        render(
            <ActionView
                {...baseProps}
                pollMetadata={{[samplePollId]: {...baseMetadata, setting_public_add_option: true}}}
            />,
        );

        expect(renderedActionIds()).toContain('addOption');
        expect(renderedActionIds()).not.toContain('endPoll');
    });

    test('should mark the answers the user has voted for', () => {
        render(<ActionView {...baseProps}/>);

        expect(screen.getByTestId('action_id1').dataset.hasVoted).toBe('true');
        expect(screen.getByTestId('action_id2').dataset.hasVoted).toBe('true');
        expect(screen.getByTestId('action_id3').dataset.hasVoted).toBe('false');

        // Management and add-option buttons are never a vote.
        expect(screen.getByTestId('resetVote').dataset.hasVoted).toBe('false');
    });

    test('should strip the vote count from the name before matching votes with setting_progress', () => {
        render(
            <ActionView
                {...baseProps}
                attachment={{
                    actions: [
                        {id: 'action_id1', name: 'answer1 (1)', type: ActionButtonType.BUTTON},
                        {id: 'action_id2', name: 'answer2 (12)', type: ActionButtonType.BUTTON},
                        {id: 'action_id3', name: 'answer3', type: ActionButtonType.BUTTON},
                    ],
                }}
                pollMetadata={{
                    [samplePollId]: {
                        ...baseMetadata,
                        voted_answers: ['answer1', 'answer3'],
                        setting_progress: true,
                    },
                }}
            />,
        );

        expect(screen.getByTestId('action_id1').dataset.hasVoted).toBe('true');
        expect(screen.getByTestId('action_id2').dataset.hasVoted).toBe('false');
        expect(screen.getByTestId('action_id3').dataset.hasVoted).toBe('true');
    });

    test('should render nothing without any actions', () => {
        const {container} = render(
            <ActionView
                {...baseProps}
                attachment={{actions: []}}
            />,
        );

        expect(container).toBeEmptyDOMElement();
    });

    test('should skip actions that are not buttons', () => {
        render(
            <ActionView
                {...baseProps}
                attachment={{
                    actions: [
                        {id: 'action_id1', name: 'answer1', type: ActionButtonType.BUTTON},
                        {id: 'action_id2', name: 'answer2', type: ActionButtonType.SELECT},
                    ],
                }}
            />,
        );

        expect(renderedActionIds()).toEqual(['action_id1']);
    });

    test('should skip actions missing an id or a name', () => {
        render(
            <ActionView
                {...baseProps}
                attachment={{
                    actions: [
                        {id: 'action_id1', name: 'answer1', type: ActionButtonType.BUTTON},
                        {id: 'action_id2', name: '', type: ActionButtonType.BUTTON},
                        {id: '', name: 'answer3', type: ActionButtonType.BUTTON},
                        {id: 'action_id4', name: 'answer4', type: ActionButtonType.BUTTON},
                    ],
                }}
            />,
        );

        expect(renderedActionIds()).toEqual(['action_id1', 'action_id4']);
    });
});
