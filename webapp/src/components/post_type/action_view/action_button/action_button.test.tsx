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
        theme: Preferences.THEMES.denim,
        hasVoted: false,
        actions: {
            voteAnswer: jest.fn(),
        },
    };

    test('should render the action name and carry its ids', () => {
        render(<ActionButton {...baseProps}/>);

        const button = screen.getByRole('button');
        expect(button).toHaveTextContent('mockMessageHtmlToComponent(mockFormatText(action_name))');
        expect(button).toHaveAttribute('data-action-id', 'action_id1');
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

    const stylesOf = (button: HTMLElement) => {
        const {backgroundColor, color} = window.getComputedStyle(button);
        return {backgroundColor, color};
    };

    // jsdom reports the user-agent default ('ButtonFace') for an unstyled button, so
    // "not coloured" means "still the default", not "no background at all".
    const UNSTYLED_BACKGROUND = 'ButtonFace';

    test('should not colour the button without a style', () => {
        render(<ActionButton {...baseProps}/>);

        expect(stylesOf(screen.getByRole('button')).backgroundColor).toBe(UNSTYLED_BACKGROUND);
    });

    // Named styles resolve against the status colours or the theme; a bare hex is used
    // as-is. A voted button takes the colour at near-full opacity with inverted text.
    test.each([
        ['default', false, 'rgba(63, 67, 80, 0.08)', 'rgb(63, 67, 80)'],
        ['default', true, 'rgba(63, 67, 80, 0.92)', 'rgb(192, 188, 175)'],
        ['primary', false, 'rgba(28, 88, 217, 0.08)', 'rgb(28, 88, 217)'],
        ['danger', false, 'rgba(210, 75, 78, 0.08)', 'rgb(210, 75, 78)'],
        ['good', false, 'rgba(51, 153, 112, 0.08)', 'rgb(51, 153, 112)'],
        ['warning', false, 'rgba(204, 143, 0, 0.08)', 'rgb(204, 143, 0)'],
        ['#0000ff', false, 'rgba(0, 0, 255, 0.08)', 'rgb(0, 0, 255)'],
    ] as [string, boolean, string, string][])(
        'should colour the button for style %s (hasVoted: %s)',
        (style, hasVoted, backgroundColor, color) => {
            render(
                <ActionButton
                    {...baseProps}
                    action={{...baseProps.action, style}}
                    hasVoted={hasVoted}
                />,
            );

            expect(stylesOf(screen.getByRole('button'))).toEqual({backgroundColor, color});
        },
    );

    test('should not colour the button for an unrecognised style', () => {
        render(
            <ActionButton
                {...baseProps}
                action={{...baseProps.action, style: 'invalid_style_value'}}
            />,
        );

        expect(stylesOf(screen.getByRole('button')).backgroundColor).toBe(UNSTYLED_BACKGROUND);
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
