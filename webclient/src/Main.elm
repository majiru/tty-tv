module Main exposing (..)

import Html exposing (..)
import WebSocket


type alias Model =
    { rows : List String
    }


init : ( Model, Cmd msg )
init =
    ( { rows = [ "initial" ] }, Cmd.none )


type Msg
    = NewMessageFromServer String


update : Msg -> Model -> ( Model, Cmd msg )
update msg model =
    case msg of
        NewMessageFromServer s ->
            let
                _ =
                    Debug.log s
            in
                ( { model | rows = model.rows ++ [ s ] }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions model =
    WebSocket.listen "/api/screen" NewMessageFromServer


view : Model -> Html msg
view model =
    let
        rowsAsHtml : List (Html msg)
        rowsAsHtml =
            List.map (\s -> div [] [ text s ]) model.rows
    in
        div [] rowsAsHtml


main : Program Never Model Msg
main =
    Html.program
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }
