import React from 'react';
import {fireEvent, render, screen} from '@testing-library/react';

import Preferences from 'mattermost-redux/constants/preferences';

import ActionButton from '@/components/post_type/action_view/action_button/action_button';

describe('components/action_button/ActionButton', () => {
    const baseProps = {
        postId: 'post_id1',
        action: {
            id: 'action_id1',
            name: 'action_name',
        },
        theme: Preferences.THEMES.denim as unknown as Record<string, string>,
        hasVoted: false,
        actions: {
            voteAnswer: jest.fn(),
        },
    };

    test('should render the action name and carry its ids', () => {
        const {asFragment} = render(<ActionButton {...baseProps}/>);

        const button = screen.getByRole('button');
        expect(button).toHaveTextContent('mockMessageHtmlToComponent(mockFormatText(action_name))');
        expect(button).toHaveAttribute('data-action-id', 'action_id1');
        expect(asFragment()).toMatchSnapshot();
    });

    test('should vote for its own action when clicked', () => {
        const voteAnswer = jest.fn();
        render(
            <ActionButton
                {...baseProps}
                actions={{voteAnswer}}
            />,
        );

        fireEvent.click(screen.getByRole('button'));

        expect(voteAnswer).toHaveBeenCalledWith('post_id1', 'action_id1');
    });

    const backgroundColorOf = (button: HTMLElement) => window.getComputedStyle(button).backgroundColor;

    // jsdom reports the user-agent default ('ButtonFace') for an unstyled button, so
    // "not coloured" means "still the default", not "no background at all".
    const UNSTYLED_BACKGROUND = 'ButtonFace';

    test('should not colour the button without a style', () => {
        render(<ActionButton {...baseProps}/>);

        expect(backgroundColorOf(screen.getByRole('button'))).toBe(UNSTYLED_BACKGROUND);
    });

    // Named styles resolve against the status colours or the theme; a bare hex is used as-is.
    // The exact colours are pinned by the snapshots.
    test.each([
        ['default', false],
        ['default', true],
        ['primary', false],
        ['danger', false],
        ['good', false],
        ['warning', false],
        ['#0000ff', false],
    ])('should colour the button for style %s (hasVoted: %s)', (style, hasVoted) => {
        const {asFragment} = render(
            <ActionButton
                {...baseProps}
                action={{...baseProps.action, style}}
                hasVoted={hasVoted}
            />,
        );

        expect(backgroundColorOf(screen.getByRole('button'))).not.toBe(UNSTYLED_BACKGROUND);
        expect(asFragment()).toMatchSnapshot();
    });

    test('should not colour the button for an unrecognised style', () => {
        render(
            <ActionButton
                {...baseProps}
                action={{...baseProps.action, style: 'invalid_style_value'}}
            />,
        );

        expect(backgroundColorOf(screen.getByRole('button'))).toBe(UNSTYLED_BACKGROUND);
    });

    test('should not throw when the theme is missing the colour a style names', () => {
        render(
            <ActionButton
                {...baseProps}
                theme={{}}
                action={{...baseProps.action, style: 'default'}}
            />,
        );

        expect(screen.getByRole('button')).toBeInTheDocument();
    });
});
